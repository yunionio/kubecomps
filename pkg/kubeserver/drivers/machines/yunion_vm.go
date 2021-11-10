package machines

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	ocapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	cloudmod "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	"yunion.io/x/kubecomps/pkg/kubeserver/models/manager"
	onecloudcli "yunion.io/x/kubecomps/pkg/utils/onecloud/client"
	"yunion.io/x/kubecomps/pkg/utils/ssh"
)

const (
	// maxPods 110 is kubelet default value
	DefaultKubeletMaxPods = 110
)

var (
	yunionVMDriver IYunionVMDriver
)

type IYunionVMDriver interface {
	RegisterHypervisor(IYunionVmHypervisor)
	GetHypervisor(hypervisor string) (IYunionVmHypervisor, error)
}

type IYunionVmHypervisor interface {
	GetHypervisor() api.ProviderType
	FindSystemDiskImage(s *mcclient.ClientSession, zoneId string) (jsonutils.JSONObject, error)
}

type sYunionVMDriver struct {
	*sBaseDriver

	hypervisorDrivers *drivers.DriverManager
}

type sYunionVMProviderDriver struct {
	*sYunionVMDriver
	provider api.ProviderType
}

func newYunionVMProviderDriver(vmDrv *sYunionVMDriver, provider api.ProviderType) models.IMachineDriver {
	return &sYunionVMProviderDriver{
		sYunionVMDriver: vmDrv,
		provider:        provider,
	}
}

func (d *sYunionVMProviderDriver) GetProvider() api.ProviderType {
	return d.provider
}

func newYunionVMDriver() *sYunionVMDriver {
	return &sYunionVMDriver{
		sBaseDriver:       newBaseDriver(),
		hypervisorDrivers: drivers.NewDriverManager(""),
	}
}

func init() {
	GetYunionVMDriver()
}

func GetYunionVMDriver() IYunionVMDriver {
	if yunionVMDriver != nil {
		return yunionVMDriver
	}
	yunionVMDriver = newYunionVMDriver()
	return yunionVMDriver
}

func (d *sYunionVMDriver) RegisterHypervisor(drv IYunionVmHypervisor) {
	d.hypervisorDrivers.Register(drv, string(drv.GetHypervisor()))
	models.RegisterMachineDriver(newYunionVMProviderDriver(
		d, drv.GetHypervisor(),
	))
}

func (d *sYunionVMDriver) GetHypervisor(hypervisor string) (IYunionVmHypervisor, error) {
	drv, err := d.hypervisorDrivers.Get(hypervisor)
	if err != nil {
		return nil, err
	}
	return drv.(IYunionVmHypervisor), nil
}

func (d *sYunionVMDriver) GetResourceType() api.MachineResourceType {
	return api.MachineResourceTypeVm
}

func (d *sYunionVMDriver) ValidateCreateData(s *mcclient.ClientSession, input *api.CreateMachineData) error {
	if input.ResourceType != api.MachineResourceTypeVm {
		return httperrors.NewInputParameterError("Invalid resource type: %q", input.ResourceType)
	}
	if len(input.ResourceId) != 0 {
		return httperrors.NewInputParameterError("Resource id must not provide")
	}

	config := input.Config.Vm

	// validate network
	if len(config.Networks) == 0 {
		return httperrors.NewNotEmptyError("Network must provide")
	}
	if len(config.Networks) != 1 {
		return httperrors.NewInputParameterError("Only 1 network can provide")
	}
	net := config.Networks[0]
	if net.Network == "" {
		return httperrors.NewNotEmptyError("Network must specified")
	}

	cli := onecloudcli.NewClientSets(s)

	netDetail, err := cli.Networks().GetDetails(net.Network)
	input.NetworkId = netDetail.Id
	if input.ZoneId == "" {
		input.ZoneId = netDetail.ZoneId
	}
	if netDetail.VpcId != input.VpcId {
		return httperrors.NewInputParameterError("Network %s int vpc %s, not in vpc %s", netDetail.Name, netDetail.Vpc, input.VpcId)
	}

	// validate zone
	zoneDetail, err := cli.Zones().GetDetails(input.ZoneId)
	if err != nil {
		return errors.Wrapf(err, "Get cloud zone %s", input.ZoneId)
	}
	if zoneDetail.Id != netDetail.ZoneId {
		return httperrors.NewInputParameterError("Network %s not in zone %s", netDetail.Name, zoneDetail.Name)
	}
	input.ZoneId = zoneDetail.Id
	if zoneDetail.CloudregionId != input.CloudregionId {
		return httperrors.NewInputParameterError("Zone %s not in cloudregion %s", zoneDetail.Name, input.CloudregionId)
	}
	config.PreferZone = input.ZoneId

	// validate sku
	if config.InstanceType != "" {
		skuDetail, err := cli.Skus().GetDetails(config.InstanceType, input.ZoneId)
		if err != nil {
			return errors.Wrapf(err, "get zone %s sku %s detail", input.ZoneId, config.InstanceType)
		}
		// config.InstanceType = skuDetail.Id
		config.VcpuCount = skuDetail.CpuCoreCount
		config.VmemSize = skuDetail.MemorySizeMB
	}

	return d.validateConfig(s, config)
}

func (d *sYunionVMDriver) findImage(s *mcclient.ClientSession, hypervisor string, zoneId string, imageId string) (string, error) {
	if imageId != "" {
		return imageId, nil
	}

	drv, err := d.GetHypervisor(hypervisor)
	if err != nil {
		return "", err
	}

	obj, err := drv.FindSystemDiskImage(s, zoneId)
	if err != nil {
		return "", err
	}

	return obj.GetString("id")
}

func (d *sYunionVMDriver) validateRootDisk(s *mcclient.ClientSession, hypervisor string, zoneId string, conf *ocapi.DiskConfig) error {
	imageId, err := d.findImage(s, hypervisor, zoneId, conf.ImageId)
	if err != nil {
		return httperrors.NewInputParameterError("Invalid kubernetes base image: %v", err)
	}
	log.Infof("Use image %s for %s", imageId, hypervisor)
	conf.ImageId = imageId
	return nil
}

func (d *sYunionVMDriver) validateConfig(s *mcclient.ClientSession, config *api.MachineCreateVMConfig) error {
	if config.VcpuCount < 2 {
		return httperrors.NewNotAcceptableError("CPU count must large than 4")
	}
	if config.VmemSize < 4096 {
		return httperrors.NewNotAcceptableError("Memory size must large than 4G")
	}

	if err := d.validateRootDisk(s, config.Hypervisor, config.PreferZone, config.Disks[0]); err != nil {
		return errors.Wrap(err, "validate root disk")
	}

	input := &ocapi.ServerCreateInput{
		ServerConfigs: &ocapi.ServerConfigs{
			PreferRegion:     config.PreferRegion,
			PreferZone:       config.PreferZone,
			PreferWire:       config.PreferWire,
			PreferHost:       config.PreferHost,
			PreferBackupHost: config.PreferBackupHost,
			Hypervisor:       config.Hypervisor,
			Disks:            config.Disks,
			Networks:         config.Networks,
			IsolatedDevices:  config.IsolatedDevices,
			InstanceType:     config.InstanceType,
		},
	}
	if config.InstanceType == "" {
		input.VmemSize = config.VmemSize
		input.VcpuCount = config.VcpuCount
	} else {
		input.VmemSize = 0
		input.VcpuCount = 0
	}
	validateData := input.JSON(input)
	ret, err := cloudmod.Servers.PerformClassAction(s, "check-create-data", validateData)
	log.Infof("check server create data: %s, ret: %s err: %v", validateData, ret, err)
	if err != nil {
		return err
	}
	return nil
}

func (d *sYunionVMDriver) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, cluster *models.SCluster, machine *models.SMachine, data *jsonutils.JSONDict) error {
	return nil
}

type ServerCreateInput struct {
	ocapi.ServerCreateInput
	SrcIpCheck  bool `json:"src_ip_check"`
	SrcMacCheck bool `json:"src_mac_check"`
}

func (d *sYunionVMDriver) getServerCreateInput(machine *models.SMachine, prepareInput *api.MachinePrepareInput) (*ServerCreateInput, error) {
	tmpFalse := false
	tmpTrue := true
	config := prepareInput.Config.Vm

	input := ServerCreateInput{
		ServerCreateInput: ocapi.ServerCreateInput{
			ServerConfigs: new(ocapi.ServerConfigs),
			VmemSize:      config.VmemSize,
			VcpuCount:     config.VcpuCount,
			AutoStart:     true,
			//EnableCloudInit: true,
		},
	}
	input.InstanceType = config.InstanceType
	if input.InstanceType != "" {
		input.VcpuCount = 0
		input.VmemSize = 0
	}

	input.Name = machine.Name
	input.DomainId = machine.DomainId
	input.ProjectId = machine.ProjectId
	input.IsSystem = &tmpTrue
	input.DisableDelete = &tmpFalse
	input.Hypervisor = config.Hypervisor
	input.Disks = config.Disks
	input.Networks = config.Networks
	input.IsolatedDevices = config.IsolatedDevices

	input.Project = machine.ProjectId
	input.PreferRegion = config.PreferRegion
	input.PreferZone = config.PreferZone
	input.PreferWire = config.PreferWire
	input.PreferHost = config.PreferHost

	input.SrcIpCheck = false
	input.SrcMacCheck = false

	return &input, nil
}

func (d *sYunionVMDriver) GetEIP(s *mcclient.ClientSession, resourceId string) (string, error) {
	return onecloudcli.NewClientSets(s).Servers().GetEIP(resourceId)
}

func (d *sYunionVMDriver) GetPrivateIP(s *mcclient.ClientSession, resourceId string) (string, error) {
	return onecloudcli.NewClientSets(s).Servers().GetPrivateIP(resourceId)
}

func (d *sYunionVMDriver) PrepareResource(
	s *mcclient.ClientSession,
	m *models.SMachine,
	data *api.MachinePrepareInput) (jsonutils.JSONObject, error) {
	// 1. create vm
	input, err := d.getServerCreateInput(m, data)
	if err != nil {
		return nil, errors.Wrap(err, "get server create input")
	}
	helper := onecloudcli.NewServerHelper(s)
	ret, err := helper.Create(s, input.JSON(input))
	if err != nil {
		log.Errorf("Create server error: %v, input disks: %#v", err, input.Disks[0])
		return nil, errors.Wrapf(err, "create server with input: %#v", input)
	}
	id, err := ret.GetString("id")
	if err != nil {
		return nil, err
	}
	m.SetHypervisor(input.Hypervisor)
	m.SetResourceId(id)

	// 2. wait vm running
	// wait server running and check service
	if err := helper.WaitRunning(id); err != nil {
		return nil, fmt.Errorf("Wait server %s running error: %v", id, err)
	}
	srvObj, err := helper.ObjectIsExists(id)
	if err != nil {
		return nil, err
	}
	/*
	 * intnalIp, err := d.GetPrivateIP(s, id)
	 * if err != nil {
	 *     return nil, err
	 * }
	 */

	srvDetail := new(ocapi.ServerDetails)
	if err := srvObj.Unmarshal(srvDetail); err != nil {
		return nil, err
	}

	// 3. prepare others resource
	// resOut, err := d.prepareResources(s, m, srvDetail.Id)
	_, err = d.prepareResources(s, m, srvDetail)
	if err != nil {
		return nil, errors.Wrapf(err, "prepare other resources for machine %s", m.GetName())
	}

	if err := d.waitSSHPortOpen(onecloudcli.NewClientSets(s), srvDetail.Id, m); err != nil {
		return nil, errors.Wrap(err, "wait ssh port open")
	}

	// 4. ssh run init script
	/*
	 * script, err := d.GetMachineInitScript(m, data, intnalIp, resOut.AddrCount, resOut.Eip)
	 * if err != nil {
	 *     return nil, errors.Wrapf(err, "get machine %s init script", m.GetName())
	 * }
	 * log.Debugf("Generate script: %s", script)
	 * output, err := d.RemoteRunScript(s, id, script)
	 * if err != nil {
	 *     return nil, errors.Wrapf(err, "output: %s", output)
	 * }
	 */

	return nil, nil
}

func (d *sYunionVMDriver) waitSSHPortOpen(cli onecloudcli.IClient, srvId string, m *models.SMachine) error {
	loginInfo, err := cli.Servers().GetSSHLoginInfo(srvId)
	if err != nil {
		return errors.Wrap(err, "Get server loginInfo")
	}
	if err := ssh.WaitRemotePortOpen(loginInfo.GetAccessIP(), 22, 30*time.Second, 10*time.Minute); err != nil {
		return errors.Wrapf(err, "remote %s ssh port can't connect", loginInfo.GetAccessIP())
	}
	return nil
}

type vmPreparedResource struct {
	AddrCount int
	Eip       string
}

func (d *sYunionVMDriver) prepareEIP(s *mcclient.ClientSession, srv *ocapi.ServerDetails) (*onecloudcli.ServerEIP, error) {
	eip, err := onecloudcli.NewServerHelper(s).CreateEIP(srv)
	if err != nil {
		return nil, errors.Wrap(err, "create eip")
	}
	return eip, nil
}

func (d *sYunionVMDriver) prepareResources(s *mcclient.ClientSession, m *models.SMachine, srv *ocapi.ServerDetails) (*vmPreparedResource, error) {
	isClassicNetwork, err := m.IsInClassicNetwork()
	if err != nil {
		return nil, errors.Wrap(err, "check is in classic network")
	}

	ret := &vmPreparedResource{
		AddrCount: DefaultKubeletMaxPods,
	}

	if !isClassicNetwork {
		eip, err := d.prepareEIP(s, srv)
		if err != nil {
			return nil, errors.Wrap(err, "prepare EIP")
		}
		ret.Eip = eip.IP
	}

	enableNativeIPAlloc, err := m.IsEnableNativeIPAlloc()
	if err != nil {
		return nil, errors.Wrap(err, "check machine's cluster is enable native IP alloc")
	}
	if !enableNativeIPAlloc {
		return ret, nil
	}

	addrCnt, err := d.getNetworkAddressCount(s, m)
	if err != nil {
		return nil, errors.Wrap(err, "getNetworkAddressCount")
	}

	ctx := context.Background()

	for i := 0; i < addrCnt; i++ {
		opt := new(api.MachineAttachNetworkAddressInput)
		if err := d.AttachNetworkAddress(ctx, s, m, opt); err != nil {
			return nil, errors.Wrapf(err, "attach network address, index:%d", i)
		}
	}

	if err := d.SyncNetworkAddress(ctx, s, m); err != nil {
		return nil, errors.Wrap(err, "sync network address")
	}
	ret.AddrCount = addrCnt

	return ret, nil
}

type ServerLoginInfo struct {
	*onecloudcli.ServerLoginInfo
	Hostname   string
	Ip         string
	PrivateKey string
}

func (d *sYunionVMDriver) RemoteRunScript(s *mcclient.ClientSession, srvId string, script string) (string, error) {
	loginInfo, err := onecloudcli.NewClientSets(s).Servers().GetSSHLoginInfo(srvId)
	if err != nil {
		return "", errors.Wrap(err, "Get server loginInfo")
	}
	if err := ssh.WaitRemotePortOpen(loginInfo.GetAccessIP(), 22, 30*time.Second, 10*time.Minute); err != nil {
		return "", errors.Wrapf(err, "remote %s ssh port can't connect", loginInfo.GetAccessIP())
	}
	return ssh.RemoteSSHBashScript(loginInfo.GetAccessIP(), 22, loginInfo.Username, loginInfo.Password, loginInfo.PrivateKey, script)
}

func (d *sYunionVMDriver) TerminateResource(session *mcclient.ClientSession, machine *models.SMachine) error {
	srvId := machine.ResourceId
	if len(srvId) == 0 {
		//return errors.Errorf("Machine resource id is empty")
		log.Warningf("Machine resource id is empty, skip clean cloud resource")
		return nil
	}

	helper := onecloudcli.NewServerHelper(session)

	enableDelete := jsonutils.NewDict()
	enableDelete.Add(jsonutils.JSONFalse, "disable_delete")
	if _, err := helper.Update(session, srvId, enableDelete); err != nil {
		if !onecloudcli.IsNotFoundError(err) {
			return errors.Wrapf(err, "enable server %s deletable", srvId)
		}
	}

	params := jsonutils.NewDict()
	params.Add(jsonutils.JSONTrue, "override_pending_delete")
	_, err := helper.DeleteWithParam(session, srvId, params, nil)
	if err != nil {
		if onecloudcli.IsNotFoundError(err) {
			return nil
		}
		return errors.Wrapf(err, "delete server %s", srvId)
	}
	err = helper.WaitDelete(srvId)
	return err
}

func (d *sYunionVMDriver) ListNetworkAddress(ctx context.Context, s *mcclient.ClientSession, m *models.SMachine) ([]*ocapi.NetworkAddressDetails, error) {
	if m.ResourceId == "" {
		return nil, httperrors.NewBadRequestError("Machine resource_id is empty")
	}

	return onecloudcli.NewClientSets(s).Servers().ListNetworkAddress(m.ResourceId)
}

func (d *sYunionVMDriver) getRemoteServer(s *mcclient.ClientSession, m *models.SMachine) (*ocapi.ServerDetails, error) {
	h := onecloudcli.NewServerHelper(s)
	return h.GetDetails(m.ResourceId)
}

func (d *sYunionVMDriver) getNetworkAddressCountByMemSize(memMb int) int {
	count := 0
	if memMb <= 1024 {
		count = 2
	} else {
		count = (memMb/1024)*4 - 1
	}

	return count
}

func (d *sYunionVMDriver) getNetworkAddressCount(s *mcclient.ClientSession, m *models.SMachine) (int, error) {
	// TODO: support server_sku and kinds of hypervisor
	srv, err := d.getRemoteServer(s, m)
	if err != nil {
		return 0, errors.Wrap(err, "get remote server")
	}

	return d.getNetworkAddressCountByMemSize(srv.VmemSize), nil
}

func (d *sYunionVMDriver) AttachNetworkAddress(ctx context.Context, s *mcclient.ClientSession, m *models.SMachine, opt *api.MachineAttachNetworkAddressInput) error {
	if m.ResourceId == "" {
		return httperrors.NewBadRequestError("Machine %s not related with remote resource", m.GetName())
	}

	return onecloudcli.NewClientSets(s).Servers().AttachNetworkAddress(m.ResourceId, opt.IPAddr)
}

type CalicoNodeAgentConfig struct {
	IPPools           CalicoNodeIPPools `json:"ipPools"`
	ProxyARPInterface string            `json:"proxyARPInterface"`
}

type CalicoNodeIPPools []CalicoNodeIPPool

type CalicoNodeIPPool struct {
	CIDR string `json:"cidr"`
}

func (d *sYunionVMDriver) getCalicoNodeAgentDeployScript(addrs []*ocapi.NetworkAddressDetails) (string, error) {
	// ref: pkg/drivers/clusters/addons/cni_calico.go
	configPath := "/var/run/calico/node-agent-config.yaml"

	config := CalicoNodeAgentConfig{
		ProxyARPInterface: "all",
		IPPools:           make([]CalicoNodeIPPool, 0),
	}
	for _, addr := range addrs {
		config.IPPools = append(config.IPPools, CalicoNodeIPPool{
			CIDR: fmt.Sprintf("%s/32", addr.IpAddr),
		})
	}

	yamlStr := jsonutils.Marshal(config).YAMLString()
	return fmt.Sprintf(`mkdir -p /var/run/calico/
cat >%s<<EOF
%s
EOF`, configPath, yamlStr), nil
}

func (d *sYunionVMDriver) SyncNetworkAddress(ctx context.Context, s *mcclient.ClientSession, m *models.SMachine) error {
	addrs, err := d.ListNetworkAddress(ctx, s, m)
	if err != nil {
		return errors.Wrap(err, "ListNetworkAddress")
	}

	script, err := d.getCalicoNodeAgentDeployScript(addrs)
	if err != nil {
		return errors.Wrap(err, "getCalicoNodeAgentDeployScript")
	}

	_, err = d.RemoteRunScript(s, m.ResourceId, script)
	return err
}

func (d *sYunionVMDriver) GetInfoFromCloud(ctx context.Context, s *mcclient.ClientSession, m *models.SMachine) (*api.CloudMachineInfo, error) {
	// get server details
	id := m.GetResourceId()
	helper := onecloudcli.NewServerHelper(s)
	srvObj, err := helper.ObjectIsExists(id)
	if err != nil {
		return nil, errors.Wrap(err, "get cloud server")
	}
	if srvObj == nil {
		return nil, errors.Wrapf(errors.ErrNotFound, "machine %s cloud server %s not found", m.GetName(), id)
	}
	srvDetail := new(ocapi.ServerDetails)
	if err := srvObj.Unmarshal(srvDetail); err != nil {
		return nil, err
	}

	// get server networks
	if m.Address == "" {
		return nil, errors.Errorf("address is empty")
	}
	nets, err := helper.ListServerNetworks(id)
	if err != nil {
		return nil, errors.Wrap(err, "list server network address")
	}
	var curNet *ocapi.SGuestnetwork
	for _, net := range nets {
		if net.IpAddr == m.Address {
			curNet = net
			break
		}
	}

	out := &api.CloudMachineInfo{
		Id:         srvDetail.Id,
		Name:       srvDetail.Name,
		Hypervisor: srvDetail.Hypervisor,
		ZoneId:     srvDetail.ZoneId,
		NetworkId:  curNet.NetworkId,
	}
	return out, nil
}

func (d *sYunionVMDriver) ValidateDeleteCondition(ctx context.Context, userCred mcclient.TokenCredential, cluster *models.SCluster, machine *models.SMachine) error {
	return cluster.GetDriver().ValidateDeleteMachines(ctx, userCred, cluster, []manager.IMachine{machine})
}
