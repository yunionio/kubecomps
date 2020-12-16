package machines

import (
	"context"
	"fmt"
	"time"

	kubeadmv1beta1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	ocapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	cloudmod "yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/machines/kubeadm"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/machines/userdata"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	"yunion.io/x/kubecomps/pkg/utils/certificates"
	onecloudcli "yunion.io/x/kubecomps/pkg/utils/onecloud/client"
	"yunion.io/x/kubecomps/pkg/utils/ssh"
)

const (
	// maxPods 110 is kubelet default value
	DefaultKubeletMaxPods = 110
)

type SYunionVMDriver struct {
	*sClusterAPIBaseDriver
}

func NewYunionVMDriver() *SYunionVMDriver {
	return &SYunionVMDriver{
		sClusterAPIBaseDriver: newClusterAPIBaseDriver(),
	}
}

func init() {
	driver := NewYunionVMDriver()
	models.RegisterMachineDriver(driver)
}

func (d *SYunionVMDriver) GetProvider() api.ProviderType {
	return api.ProviderTypeOnecloud
}

func (d *SYunionVMDriver) GetResourceType() api.MachineResourceType {
	return api.MachineResourceTypeVm
}

func (d *SYunionVMDriver) ValidateCreateData(s *mcclient.ClientSession, input *api.CreateMachineData) error {
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
	netObj, err := cloudmod.Networks.Get(s, net.Network, nil)
	if err != nil {
		return errors.Wrapf(err, "Get cloud network %s", net.Network)
	}
	netDetail := new(ocapi.NetworkDetails)
	if err := netObj.Unmarshal(netDetail); err != nil {
		return errors.Wrap(err, "Unmarshal network")
	}
	input.NetworkId = netDetail.Id
	if input.ZoneId == "" {
		input.ZoneId = netDetail.ZoneId
	}
	if netDetail.VpcId != input.VpcId {
		return httperrors.NewInputParameterError("Network %s int vpc %s, not in vpc %s", netDetail.Name, netDetail.Vpc, input.VpcId)
	}

	// validate zone
	zoneObj, err := cloudmod.Zones.Get(s, input.ZoneId, nil)
	if err != nil {
		return errors.Wrapf(err, "Get cloud zone %s", input.ZoneId)
	}
	zoneDetail := new(ocapi.ZoneDetails)
	if err := zoneObj.Unmarshal(zoneDetail); err != nil {
		return errors.Wrap(err, "Unmarshal zone")
	}
	if zoneDetail.Id != netDetail.ZoneId {
		return httperrors.NewInputParameterError("Network %s not in zone %s", netDetail.Name, zoneDetail.Name)
	}
	input.ZoneId = zoneDetail.Id
	if zoneDetail.CloudregionId != input.CloudregionId {
		return httperrors.NewInputParameterError("Zone %s not in cloudregion %s", zoneDetail.Name, input.CloudregionId)
	}

	return d.validateConfig(s, config)
}

func (d *SYunionVMDriver) validateConfig(s *mcclient.ClientSession, config *api.MachineCreateVMConfig) error {
	if config.VcpuCount < 4 {
		return httperrors.NewNotAcceptableError("CPU count must large than 4")
	}
	if config.VmemSize < 4096 {
		return httperrors.NewNotAcceptableError("Memory size must large than 4G")
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
		},
		VmemSize:  config.VmemSize,
		VcpuCount: config.VcpuCount,
	}
	validateData := input.JSON(input)
	ret, err := cloudmod.Servers.PerformClassAction(s, "check-create-data", validateData)
	log.Infof("check server create data: %s, ret: %s err: %v", validateData, ret, err)
	if err != nil {
		return err
	}
	return nil
}

func (d *SYunionVMDriver) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, cluster *models.SCluster, machine *models.SMachine, data *jsonutils.JSONDict) error {
	return d.sClusterAPIBaseDriver.PostCreate(ctx, userCred, cluster, machine, data)
}

type ServerCreateInput struct {
	ocapi.ServerCreateInput
	SrcIpCheck  bool `json:"src_ip_check"`
	SrcMacCheck bool `json:"src_mac_check"`
}

func (d *SYunionVMDriver) getServerCreateInput(machine *models.SMachine, prepareInput *api.MachinePrepareInput) (*ServerCreateInput, error) {
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

func GetDefaultDockerConfig(input *api.DockerConfig) *api.DockerConfig {
	if input.Graph == "" {
		input.Graph = api.DefaultDockerGraphDir
	}
	if len(input.RegistryMirrors) == 0 {
		input.RegistryMirrors = []string{
			api.DefaultDockerRegistryMirror1,
			api.DefaultDockerRegistryMirror2,
			api.DefaultDockerRegistryMirror3,
		}
	}
	input.Bridge = "none"
	input.Iptables = false
	input.LiveRestore = true
	if len(input.ExecOpts) == 0 {
		//ExecOpts:           []string{"native.cgroupdriver=systemd"},
		input.ExecOpts = []string{"native.cgroupdriver=cgroupfs"}
	}
	if input.LogDriver == "" {
		input.LogDriver = "json-file"
		input.LogOpts = api.DockerConfigLogOpts{
			MaxSize: "100m",
		}
	}
	if input.StorageDriver == "" {
		input.StorageDriver = "overlay2"
	}
	return input
}

func (d *SYunionVMDriver) GetMachineInitScript(
	machine *models.SMachine,
	data *api.MachinePrepareInput,
	interalIP string,
	maxPods int,
	eIP string,
) (string, error) {
	var initScript string
	var err error

	if maxPods == 0 {
		maxPods = DefaultKubeletMaxPods
	}

	caCertHash, err := certificates.GenerateCertificateHash(data.CAKeyPair.Cert)
	if err != nil {
		return "", err
	}

	cluster, err := machine.GetCluster()
	if err != nil {
		return "", err
	}

	imageRepo := data.Config.ImageRepository
	kubeletExtraArgs := map[string]string{
		"cgroup-driver":             "cgroupfs",
		"read-only-port":            "10255",
		"pod-infra-container-image": fmt.Sprintf("%s/pause-amd64:3.1", imageRepo.Url),
		"feature-gates":             "CSIPersistentVolume=true,KubeletPluginsWatcher=true,VolumeScheduling=true",
		"eviction-hard":             "memory.available<100Mi,nodefs.available<2Gi,nodefs.inodesFree<5%",
		"max-pods":                  fmt.Sprintf("%d", maxPods),
	}
	dockerConfig := GetDefaultDockerConfig(data.Config.DockerConfig)
	switch data.Role {
	case api.RoleTypeControlplane:
		if data.BootstrapToken != "" {
			log.Infof("Allowing a machine to join the control plane")
			apiServerEndpoint, err := cluster.GetAPIServerInternalEndpoint()
			if err != nil {
				return "", err
			}
			updatedJoinConfiguration := kubeadm.SetJoinNodeConfigurationOverrides(caCertHash, data.BootstrapToken, apiServerEndpoint, nil, machine.Name)
			updatedJoinConfiguration = kubeadm.SetControlPlaneJoinConfigurationOverrides(updatedJoinConfiguration, interalIP)
			initScript, err = userdata.JoinControlplaneConfig{
				DockerConfiguration: dockerConfig,
				CACert:              string(data.CAKeyPair.Cert),
				CAKey:               string(data.CAKeyPair.Key),
				EtcdCACert:          string(data.EtcdCAKeyPair.Cert),
				EtcdCAKey:           string(data.EtcdCAKeyPair.Key),
				FrontProxyCACert:    string(data.FrontProxyCAKeyPair.Cert),
				FrontProxyCAKey:     string(data.FrontProxyCAKeyPair.Key),
				SaCert:              string(data.SAKeyPair.Cert),
				SaKey:               string(data.SAKeyPair.Key),
				JoinConfiguration:   updatedJoinConfiguration,
			}.ToScript()
			if err != nil {
				return "", errors.Wrap(err, "generate join controlplane script")
			}
		} else {
			log.Infof("Machine is the first control plane machine for the cluster")
			if !data.CAKeyPair.HasCertAndKey() {
				return "", errors.Error("failed to run controlplane, missing CAPrivateKey")
			}

			clusterConfiguration, err := kubeadm.SetClusterConfigurationOverrides(cluster, nil, interalIP, eIP)
			if err != nil {
				return "", errors.Wrap(err, "SetClusterConfigurationOverrides")
			}
			clusterConfiguration.APIServer.ExtraArgs = map[string]string{
				"cloud-provider": "external",
				"feature-gates":  "CSIPersistentVolume=true",
				//"runtime-config": "storage.k8s.io/v1alpha1=true,admissionregistration.k8s.io/v1alpha1=true,settings.k8s.io/v1alpha1=true",
			}
			clusterConfiguration.ControllerManager.ExtraArgs = map[string]string{
				"cloud-provider": "external",
				"feature-gates":  "CSIPersistentVolume=true",
			}
			clusterConfiguration.Scheduler.ExtraArgs = map[string]string{
				"feature-gates": "CSIPersistentVolume=true",
			}
			clusterConfiguration.ImageRepository = imageRepo.Url

			initConfiguration := kubeadm.SetInitConfigurationOverrides(&kubeadmv1beta1.InitConfiguration{
				NodeRegistration: kubeadmv1beta1.NodeRegistrationOptions{
					KubeletExtraArgs: kubeletExtraArgs,
				},
			}, machine.Name)

			kubeProxyConfiguration := kubeadm.SetKubeProxyConfigurationOverrides(nil, cluster.GetServiceCidr())

			initScript, err = userdata.InitNodeConfig{
				DockerConfiguration:    dockerConfig,
				CACert:                 string(data.CAKeyPair.Cert),
				CAKey:                  string(data.CAKeyPair.Key),
				EtcdCACert:             string(data.EtcdCAKeyPair.Cert),
				EtcdCAKey:              string(data.EtcdCAKeyPair.Key),
				FrontProxyCACert:       string(data.FrontProxyCAKeyPair.Cert),
				FrontProxyCAKey:        string(data.FrontProxyCAKeyPair.Key),
				SaCert:                 string(data.SAKeyPair.Cert),
				SaKey:                  string(data.SAKeyPair.Key),
				ClusterConfiguration:   clusterConfiguration,
				InitConfiguration:      initConfiguration,
				KubeProxyConfiguration: kubeProxyConfiguration,
			}.ToScript()

			if err != nil {
				return "", err
			}
		}
	case api.RoleTypeNode:
		apiServerEndpoint, err := cluster.GetAPIServerInternalEndpoint()
		if err != nil {
			return "", err
		}
		joinConfiguration := kubeadm.SetJoinNodeConfigurationOverrides(caCertHash, data.BootstrapToken, apiServerEndpoint, nil, machine.Name)
		joinConfiguration.NodeRegistration.KubeletExtraArgs = kubeletExtraArgs
		initScript, err = userdata.JoinNodeConfig{
			DockerConfiguration: dockerConfig,
			JoinConfiguration:   joinConfiguration,
		}.ToScript()
		if err != nil {
			return "", err
		}
	}
	return initScript, nil
}

func (d *SYunionVMDriver) PrepareResource(
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
	intnalIp, err := d.GetPrivateIP(s, id)
	if err != nil {
		return nil, err
	}

	srvDetail := new(ocapi.ServerDetails)
	if err := srvObj.Unmarshal(srvDetail); err != nil {
		return nil, err
	}

	// 3. prepare others resource
	resOut, err := d.prepareResources(s, m, srvDetail.Id)
	if err != nil {
		return nil, errors.Wrapf(err, "prepare other resources for machine %s", m.GetName())
	}

	// 4. ssh run init script
	script, err := d.GetMachineInitScript(m, data, intnalIp, resOut.AddrCount, resOut.Eip)
	if err != nil {
		return nil, errors.Wrapf(err, "get machine %s init script", m.GetName())
	}
	log.Debugf("Generate script: %s", script)
	output, err := d.RemoteRunScript(s, id, script)
	if err != nil {
		return nil, errors.Wrapf(err, "output: %s", output)
	}

	return nil, nil
}

type vmPreparedResource struct {
	AddrCount int
	Eip       string
}

func (d *SYunionVMDriver) prepareEIP(s *mcclient.ClientSession, srvId string) (*onecloudcli.ServerEIP, error) {
	eip, err := onecloudcli.NewServerHelper(s).CreateEIP(srvId)
	if err != nil {
		return nil, errors.Wrap(err, "create eip")
	}
	return eip, nil
}

func (d *SYunionVMDriver) prepareResources(s *mcclient.ClientSession, m *models.SMachine, srvId string) (*vmPreparedResource, error) {
	isClassicNetwork, err := m.IsInClassicNetwork()
	if err != nil {
		return nil, errors.Wrap(err, "check is in classic network")
	}

	ret := &vmPreparedResource{
		AddrCount: DefaultKubeletMaxPods,
	}

	if !isClassicNetwork {
		eip, err := d.prepareEIP(s, srvId)
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
	Ip         string
	PrivateKey string
}

func (d *SYunionVMDriver) GetServerLoginInfo(s *mcclient.ClientSession, srvId string) (*ServerLoginInfo, error) {
	helper := onecloudcli.NewServerHelper(s)
	privateKey, err := onecloudcli.GetCloudSSHPrivateKey(s)
	if err != nil {
		return nil, errors.Wrapf(err, "GetCloudSSHPrivateKey")
	}
	detail, err := helper.GetDetails(srvId)
	if err != nil {
		return nil, errors.Wrapf(err, "Get server detail")
	}
	ip := detail.Eip
	if ip == "" {
		ip, err = d.GetPrivateIP(s, srvId)
		if err != nil {
			return nil, errors.Wrapf(err, "Get server %q PrivateIP", srvId)
		}
	}
	loginInfo, err := helper.GetLoginInfo(srvId)
	if err != nil {
		return nil, errors.Wrapf(err, "Get server %q loginInfo", srvId)
	}
	return &ServerLoginInfo{
		ServerLoginInfo: loginInfo,
		Ip:              ip,
		PrivateKey:      privateKey,
	}, nil
}

func (d *SYunionVMDriver) RemoteRunScript(s *mcclient.ClientSession, srvId string, script string) (string, error) {
	loginInfo, err := d.GetServerLoginInfo(s, srvId)
	if err != nil {
		return "", errors.Wrap(err, "Get server loginInfo")
	}
	if err := ssh.WaitRemotePortOpen(loginInfo.Ip, 22, 30*time.Second, 10*time.Minute); err != nil {
		return "", errors.Wrapf(err, "remote %s ssh port can't connect", loginInfo.Ip)
	}
	return ssh.RemoteSSHBashScript(loginInfo.Ip, 22, loginInfo.Username, loginInfo.Password, loginInfo.PrivateKey, script)
}

func (d *SYunionVMDriver) RemoteRunCmd(s *mcclient.ClientSession, srvId string, cmd string) (string, error) {
	loginInfo, err := d.GetServerLoginInfo(s, srvId)
	if err != nil {
		return "", errors.Wrap(err, "Get server loginInfo")
	}
	if err := ssh.WaitRemotePortOpen(loginInfo.Ip, 22, 30*time.Second, 10*time.Minute); err != nil {
		return "", errors.Wrapf(err, "remote %s ssh port can't connect", loginInfo.Ip)
	}
	return ssh.RemoteSSHCommand(loginInfo.Ip, 22, loginInfo.Username, loginInfo.Password, loginInfo.PrivateKey, cmd)
}

func (d *SYunionVMDriver) TerminateResource(session *mcclient.ClientSession, machine *models.SMachine) error {
	srvId := machine.ResourceId
	if len(srvId) == 0 {
		//return errors.Errorf("Machine resource id is empty")
		log.Warningf("Machine resource id is empty, skip clean cloud resource")
		return nil
	}
	if len(machine.Address) != 0 && !machine.IsFirstNode() {
		_, err := d.RemoteRunCmd(session, srvId, "sudo kubeadm reset -f")
		if err != nil {
			//return errors.Wrap(err, "kubeadm reset failed")
			log.Errorf("kubeadm reset failed: %v", err)
		}
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

func (d *SYunionVMDriver) ListServerNetworks(session *mcclient.ClientSession, id string) ([]*ocapi.SGuestnetwork, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.JSONTrue, "system")
	params.Add(jsonutils.JSONTrue, "admin")
	ret, err := cloudmod.Servernetworks.ListDescendent(session, id, params)
	if err != nil {
		return nil, err
	}
	if len(ret.Data) == 0 {
		return nil, errors.Errorf("Not found networks by id: %s", id)
	}
	objs := make([]*ocapi.SGuestnetwork, 0)
	for _, data := range ret.Data {
		obj := new(ocapi.SGuestnetwork)
		if err := data.Unmarshal(obj); err != nil {
			return nil, err
		}
		objs = append(objs, obj)
	}
	return objs, nil
}

func (d *SYunionVMDriver) GetPrivateIP(s *mcclient.ClientSession, id string) (string, error) {
	nets, err := d.ListServerNetworks(s, id)
	if err != nil {
		return "", errors.Wrap(err, "list server networks")
	}
	return nets[0].IpAddr, nil
}

func (d *SYunionVMDriver) GetEIP(s *mcclient.ClientSession, id string) (string, error) {
	obj, err := onecloudcli.NewClientSets(s).Servers().GetDetails(id)
	if err != nil {
		return "", errors.Wrap(err, "get cloud server details")
	}
	if obj.Eip == "" {
		return "", errors.Errorf("server %s not found eip", id)
	}
	return obj.Eip, nil
}

func (d *SYunionVMDriver) ListNetworkAddress(ctx context.Context, s *mcclient.ClientSession, m *models.SMachine) ([]*ocapi.NetworkAddressDetails, error) {
	if m.ResourceId == "" {
		return nil, httperrors.NewBadRequestError("Machine resource_id is empty")
	}

	input := new(ocapi.NetworkAddressListInput)
	zeroLimit := 0
	input.Limit = &zeroLimit
	input.GuestId = []string{m.ResourceId}
	ret, err := cloudmod.NetworkAddresses.List(s, input.JSON(input))
	if err != nil {
		return nil, err
	}

	objs := make([]*ocapi.NetworkAddressDetails, 0)
	for _, obj := range ret.Data {
		out := new(ocapi.NetworkAddressDetails)
		if err := obj.Unmarshal(out); err != nil {
			return nil, err
		}
		objs = append(objs, out)
	}

	return objs, nil
}

func (d *SYunionVMDriver) getRemoteServer(s *mcclient.ClientSession, m *models.SMachine) (*ocapi.ServerDetails, error) {
	h := onecloudcli.NewServerHelper(s)
	return h.GetDetails(m.ResourceId)
}

func (d *SYunionVMDriver) getNetworkAddressCountByMemSize(memMb int) int {
	count := 0
	if memMb <= 1024 {
		count = 2
	} else {
		count = (memMb/1024)*4 - 1
	}

	return count
}

func (d *SYunionVMDriver) getNetworkAddressCount(s *mcclient.ClientSession, m *models.SMachine) (int, error) {
	// TODO: support server_sku and kinds of hypervisor
	srv, err := d.getRemoteServer(s, m)
	if err != nil {
		return 0, errors.Wrap(err, "get remote server")
	}

	return d.getNetworkAddressCountByMemSize(srv.VmemSize), nil
}

func (d *SYunionVMDriver) AttachNetworkAddress(ctx context.Context, s *mcclient.ClientSession, m *models.SMachine, opt *api.MachineAttachNetworkAddressInput) error {
	if m.ResourceId == "" {
		return httperrors.NewBadRequestError("Machine %s not related with remote resource", m.GetName())
	}

	rInput := &ocapi.NetworkAddressCreateInput{
		GuestId: m.ResourceId,
		// TODO: support specify network index
		// always use first network currently
		ParentType:        ocapi.NetworkAddressParentTypeGuestnetwork,
		GuestnetworkIndex: 0,
		Type:              ocapi.NetworkAddressTypeSubIP,
		IPAddr:            opt.IPAddr,
	}

	if _, err := cloudmod.NetworkAddresses.Create(s, jsonutils.Marshal(rInput)); err != nil {
		return errors.Wrap(err, "Attach network address")
	}

	return nil
}

type CalicoNodeAgentConfig struct {
	IPPools           CalicoNodeIPPools `json:"ipPools"`
	ProxyARPInterface string            `json:"proxyARPInterface"`
}

type CalicoNodeIPPools []CalicoNodeIPPool

type CalicoNodeIPPool struct {
	CIDR string `json:"cidr"`
}

func (d *SYunionVMDriver) getCalicoNodeAgentDeployScript(addrs []*ocapi.NetworkAddressDetails) (string, error) {
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

func (d *SYunionVMDriver) SyncNetworkAddress(ctx context.Context, s *mcclient.ClientSession, m *models.SMachine) error {
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

func (d *SYunionVMDriver) GetInfoFromCloud(ctx context.Context, s *mcclient.ClientSession, m *models.SMachine) (*api.CloudMachineInfo, error) {
	// get server details
	id := m.GetResourceId()
	helper := onecloudcli.NewServerHelper(s)
	srvObj, err := helper.ObjectIsExists(id)
	if err != nil {
		return nil, errors.Wrap(err, "get cloud server")
	}
	srvDetail := new(ocapi.ServerDetails)
	if err := srvObj.Unmarshal(srvDetail); err != nil {
		return nil, err
	}

	// get server networks
	if m.Address == "" {
		return nil, errors.Errorf("address is empty")
	}
	nets, err := d.ListServerNetworks(s, id)
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
