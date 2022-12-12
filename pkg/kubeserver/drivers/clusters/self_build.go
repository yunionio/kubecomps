package clusters

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/sync/errgroup"
	"k8s.io/client-go/rest"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/constants"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/clusters/addons"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/clusters/kubespray"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/machines"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	"yunion.io/x/kubecomps/pkg/kubeserver/models/manager"
	"yunion.io/x/kubecomps/pkg/kubeserver/options"
	onecloudcli "yunion.io/x/kubecomps/pkg/utils/onecloud/client"
	"yunion.io/x/kubecomps/pkg/utils/rand"

	// "yunion.io/x/kubecomps/pkg/utils/registry"
	"yunion.io/x/kubecomps/pkg/utils/ssh"
)

// iSelfBuildDriver create k8s cluster resources through onecloud
type iSelfBuildDriver interface {
	models.IClusterDriverMethods
}

type selfBuildClusterDriver struct {
	iSelfBuildDriver
	provider     api.ProviderType
	resourceType api.ClusterResourceType
}

func registerSelfBuildClusterDriver(provider api.ProviderType) {
	registerClusterDriver(
		newSelfBuildClusterDriver(
			provider,
			// currently only support vm instance
			api.ClusterResourceTypeGuest),
	)
}

func newSelfBuildClusterDriver(provider api.ProviderType, resourceType api.ClusterResourceType) models.IClusterDriver {
	drv := &selfBuildClusterDriver{
		provider:     provider,
		resourceType: resourceType,
	}
	drv.iSelfBuildDriver = newSelfBuildDriver(drv)
	return drv
}

func (d *selfBuildClusterDriver) GetMode() api.ModeType {
	return api.ModeTypeSelfBuild
}

func (d *selfBuildClusterDriver) GetProvider() api.ProviderType {
	return d.provider
}

func (d *selfBuildClusterDriver) GetResourceType() api.ClusterResourceType {
	return d.resourceType
}

func (d *selfBuildClusterDriver) GetK8sVersions() []string {
	return []string{
		constants.K8S_VERSION_1_17_0,
		constants.K8S_VERSION_1_20_0,
	}
}

func (d *selfBuildClusterDriver) PreCheck(s *mcclient.ClientSession, data jsonutils.JSONObject) (*api.ClusterPreCheckResp, error) {
	mDrv, err := machines.GetYunionVMDriver().GetHypervisor(string(d.provider))
	if err != nil {
		return nil, err
	}

	zoneId := jsonutils.GetAnyString(data, []string{"zone", "zone_id"})

	ret := &api.ClusterPreCheckResp{
		Pass: true,
	}
	if _, err := mDrv.FindSystemDiskImage(s, zoneId); err != nil {
		log.Errorf("FindSystemDiskImage for %s error: %v", mDrv.GetHypervisor(), err)
		ret.Pass = false
		ret.ImageError = err.Error()
	}
	return ret, nil
}

func newSelfBuildDriver(driver models.IClusterDriver) iSelfBuildDriver {
	return &selfBuildDriver{
		driver: driver,
	}
}

type selfBuildDriver struct {
	driver models.IClusterDriver
}

func (c selfBuildDriver) NeedCreateMachines() bool {
	return true
}

func (c *selfBuildDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.ClusterCreateInput) error {
	// validate cloud region and vpc
	s, err := models.GetClusterManager().GetSession()
	if err != nil {
		return errors.Wrap(err, "get cloud session")
	}
	helper := onecloudcli.NewClientSets(s)
	if input.VpcId == "" {
		input.VpcId = computeapi.DEFAULT_VPC_ID
	}
	vpc, err := helper.Vpcs().GetDetails(input.VpcId)
	if err != nil {
		return errors.Wrap(err, "get cloud vpc")
	}
	input.VpcId = vpc.Id
	if input.CloudregionId == "" {
		input.CloudregionId = vpc.CloudregionId
	}

	if input.ProjectDomainId != "" {
		dObj, err := db.TenantCacheManager.FetchDomainByIdOrName(ctx, input.ProjectDomainId)
		if err != nil {
			return errors.Wrapf(err, "find domain %s", input.ProjectDomainId)
		}
		_, err = db.TenantCacheManager.FindFirstProjectOfDomain(ctx, input.ProjectDomainId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return httperrors.NewNotFoundError("not found projects in domain %s(%s)", dObj.GetName(), input.ProjectDomainId)
			}
			return errors.Wrapf(err, "find projects of domain %s", input.ProjectDomainId)
		}
	}

	cloudregion, err := helper.Cloudregions().GetDetails(input.CloudregionId)
	if err != nil {
		return errors.Wrap(err, "get cloud region")
	}
	input.CloudregionId = cloudregion.Id
	if vpc.CloudregionId != input.CloudregionId {
		return httperrors.NewInputParameterError("Vpc %s not in cloud region %s", vpc.Name, cloudregion.Name)
	}

	ms := input.Machines
	controls, _ := models.GroupCreateMachineDatas("", ms)
	if len(controls) == 0 {
		return httperrors.NewInputParameterError("No controlplane nodes")
	}
	cnts := models.GetClusterManager().GetAllowedControlplanceCount()
	if len(controls)%2 == 0 {
		return httperrors.NewInputParameterError("The number of %d controlplane nodes is not odd, should be within %v", len(controls), cnts)
	}
	allowCnt := false
	for _, cnt := range cnts {
		if len(controls) == cnt {
			allowCnt = true
			break
		}
	}
	if !allowCnt {
		return httperrors.NewInputParameterError("The number of controlplane nodes should be within %v, you input %d", cnts, len(controls))
	}

	ctx = context.WithValue(ctx, "VmNamePrefix", strings.ToLower(input.Name))
	info := &api.ClusterMachineCommonInfo{
		CloudregionId: input.CloudregionId,
		VpcId:         input.VpcId,
	}
	imageRepo := input.ImageRepository
	if err := c.ValidateCreateMachines(ctx, userCred, nil, info, imageRepo, ms); err != nil {
		return err
	}
	input.Machines = ms

	return nil
}

func getClusterMachineIndexs(cluster *models.SCluster, role string, count int) ([]int, error) {
	if count == 0 {
		return nil, nil
	}
	orderGen := func(count int) []int {
		ret := make([]int, 0)
		for i := 0; i < count; i++ {
			ret = append(ret, i)
		}
		return ret
	}
	if cluster == nil {
		return orderGen(count), nil
	}
	ms, err := cluster.GetMachinesByRole(role)
	if err != nil {
		return nil, errors.Wrapf(err, "Get machines by role %s", role)
	}
	idxs := make(map[int]bool)
	for _, m := range ms {
		name := m.GetName()
		parts := strings.Split(name, "-")
		if len(parts) == 0 {
			continue
		}
		idxStr := parts[len(parts)-1]
		idx, err := strconv.Atoi(idxStr)
		if err != nil {
			log.Errorf("Invalid machine name: %s", name)
			continue
		}
		idxs[idx] = true
	}
	if len(idxs) == 0 {
		return orderGen(count), nil
	}

	ret := make([]int, 0)

	for i := 0; i < count; i++ {
		for idx := 0; ; idx++ {
			_, ok := idxs[idx]
			if !ok {
				ret = append(ret, idx)
				idxs[idx] = true
				break
			}
		}
	}
	return ret, nil
}

func generateVMName(cluster, role, randStr string, idx int) string {
	return fmt.Sprintf("%s-%s-%s-%d", cluster, role, randStr, idx)
}

func (d *selfBuildDriver) ValidateCreateMachines(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	cluster *models.SCluster,
	info *api.ClusterMachineCommonInfo,
	imageRepo *api.ImageRepository,
	data []*api.CreateMachineData,
) error {
	controls, nodes, err := baseValidateCreateMachines(ctx, userCred, cluster, data)
	if err != nil {
		return err
	}

	cloudregionId := info.CloudregionId
	vpcId := info.VpcId

	var namePrefix string
	if cluster == nil {
		ret := ctx.Value("VmNamePrefix")
		if ret == nil {
			return errors.Error("VmNamePrefix not in context")
		}
		namePrefix = ret.(string)
		imageRepo = models.ClusterManager.GetImageRepository(imageRepo)
	} else {
		namePrefix = cluster.GetName()
		imageRepo, err = cluster.GetImageRepository()
		if err != nil {
			return errors.Wrap(err, "get cluster image repo")
		}
	}

	session, err := models.ClusterManager.GetSession()
	if err != nil {
		return err
	}
	randStr := rand.String(4)
	controlIdxs, err := getClusterMachineIndexs(cluster, api.RoleTypeControlplane, len(controls))
	if err != nil {
		return httperrors.NewNotAcceptableError("Generate controlplane machines name: %v", err)
	}
	for idx, m := range controls {
		if len(m.Name) == 0 {
			m.Name = generateVMName(namePrefix, m.Role, randStr, controlIdxs[idx])
		}
		if err := d.applyMachineCreateConfig(session, m, cloudregionId, vpcId); err != nil {
			return httperrors.NewInputParameterError("Apply controlplane vm config: %v", err)
		}
	}
	nodeIdxs, err := getClusterMachineIndexs(cluster, api.RoleTypeNode, len(nodes))
	if err != nil {
		return httperrors.NewNotAcceptableError("Generate node machines name: %v", err)
	}
	for idx, m := range nodes {
		if len(m.Name) == 0 {
			m.Name = generateVMName(namePrefix, m.Role, randStr, nodeIdxs[idx])
		}
		if err := d.applyMachineCreateConfig(session, m, cloudregionId, vpcId); err != nil {
			return httperrors.NewInputParameterError("Apply node vm config: %v", err)
		}
	}
	privateKey, err := onecloudcli.GetCloudSSHPrivateKey(session)
	if err != nil {
		return errors.Wrapf(err, "failed to get cloud ssh privateKey")
	}
	var errgrp errgroup.Group
	for _, m := range data {
		tmp := m
		errgrp.Go(func() error {
			if err := d.validateCreateMachine(session, privateKey, tmp); err != nil {
				return err
			}
			return nil
		})
	}
	if err := errgrp.Wait(); err != nil {
		return err
	}
	return nil
}

func (d *selfBuildDriver) applyMachineCreateConfig(s *mcclient.ClientSession, m *api.CreateMachineData, cloudregionId, vpcId string) error {
	m.CloudregionId = cloudregionId
	m.VpcId = vpcId

	if m.Config == nil {
		m.Config = new(api.MachineCreateConfig)
	}
	if m.Config.Vm == nil {
		m.Config.Vm = new(api.MachineCreateVMConfig)
	}
	config := m.Config.Vm
	if config.Hypervisor == "" {
		config.Hypervisor = computeapi.HYPERVISOR_KVM
	}
	if config.VmemSize <= 0 {
		config.VmemSize = api.DefaultVMMemSize
	}
	if config.VcpuCount <= 0 {
		config.VcpuCount = api.DefaultVMCPUCount
	}
	if config.VcpuCount < api.DefaultVMCPUCount {
		return errors.Errorf("cpu count less than %d", api.DefaultVMCPUCount)
	}
	rootDisk := &computeapi.DiskConfig{
		SizeMb: api.DefaultVMRootDiskSize,
	}
	restDisks := []*computeapi.DiskConfig{}
	if len(config.Disks) >= 1 {
		rootDisk = config.Disks[0]
		restDisks = config.Disks[1:]
	}
	config.Disks = []*computeapi.DiskConfig{rootDisk}
	config.Disks = append(config.Disks, restDisks...)
	config.PreferRegion = m.CloudregionId
	return nil
}

func (d *selfBuildDriver) validateCreateMachine(s *mcclient.ClientSession, privateKey string, m *api.CreateMachineData) error {
	if err := models.ValidateRole(m.Role); err != nil {
		return err
	}
	if m.ResourceType != api.MachineResourceTypeVm {
		return httperrors.NewInputParameterError("Invalid resource type: %q", m.ResourceType)
	}
	if len(m.ResourceId) != 0 {
		return httperrors.NewInputParameterError("ResourceId can't be specify")
	}
	mDrv := GetMachineDriver(d.driver, api.MachineResourceType(m.ResourceType))
	if err := mDrv.ValidateCreateData(s, m); err != nil {
		return err
	}
	return nil
}

func (d *selfBuildDriver) GetUsableInstances(s *mcclient.ClientSession) ([]api.UsableInstance, error) {
	return nil, httperrors.NewInputParameterError("Can't get UsableInstances")
}

func (d *selfBuildDriver) RequestDeployMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *models.SCluster, action api.ClusterDeployAction, ms []manager.IMachine, task taskman.ITask) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		return nil, d.requestDeployMachines(ctx, userCred, cluster, action, ms)
	})

	return nil
}

func (d *selfBuildDriver) requestDeployMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *models.SCluster, action api.ClusterDeployAction, ms []manager.IMachine) error {
	s, err := models.GetClusterManager().GetSession()
	if err != nil {
		return errors.Wrap(err, "get onecloud client session")
	}
	cli := onecloudcli.NewClientSets(s)
	return d.deployCluster(ctx, cli, cluster, action, ms)
}

func (d *selfBuildDriver) GetKubesprayVars(cluster *models.SCluster) (*kubespray.KubesprayRunVars, error) {
	extraConf, err := cluster.GetExtraConfig()
	if err != nil {
		return nil, errors.Wrap(err, "get extra config")
	}
	return &kubespray.KubesprayRunVars{
		KubesprayVars: d.withKubespray(cluster.GetVersion(), extraConf),
	}, nil
}

func (d *selfBuildDriver) GetKubesprayInventory(
	vars *kubespray.KubesprayRunVars,
	cli onecloudcli.IClient,
	cluster *models.SCluster,
	ms []manager.IMachine) ([]*kubespray.KubesprayInventoryHost, error) {
	var errgrp errgroup.Group
	hosts := make([]*kubespray.KubesprayInventoryHost, len(ms))

	for idx := range ms {
		// https://golang.org/doc/faq#closures_and_goroutines
		idx := idx
		m := ms[idx]
		errgrp.Go(func() error {
			loginInfo, err := cli.Servers().GetSSHLoginInfo(m.GetResourceId())
			if err != nil {
				return errors.Wrapf(err, "get server %s loginInfo", m.GetName())
			}

			accessIP := loginInfo.GetAccessIP()
			roles := make([]kubespray.KubesprayNodeRole, 0)
			if m.IsControlplane() {
				roles = append(roles,
					kubespray.KubesprayNodeRoleMaster,
					kubespray.KubesprayNodeRoleControlPlane,
					// controlplane should always set as etcd member
					kubespray.KubesprayNodeRoleEtcd)
				vars.SupplementaryAddresses = append(vars.SupplementaryAddresses, accessIP)
			}
			roles = append(roles, kubespray.KubesprayNodeRoleNode)

			host, err := kubespray.NewKubesprayInventoryHost(loginInfo.Hostname, accessIP, loginInfo.Username, loginInfo.Password, roles...)
			if err != nil {
				return errors.Wrapf(err, "new kubespray inventory host for machine %s", m.GetName())
			}
			if loginInfo.PrivateKey != "" {
				if err := host.SetPrivateKey([]byte(loginInfo.PrivateKey)); err != nil {
					return errors.Wrapf(err, "set private key for host %s", m.GetName())
				}
			}

			hosts[idx] = host
			return nil
		})
	}
	if err := errgrp.Wait(); err != nil {
		return nil, err
	}

	return hosts, nil
}

func (d *selfBuildDriver) GetKubesprayConfig(ctx context.Context, cluster *models.SCluster) (*api.ClusterKubesprayConfig, error) {
	s, err := models.GetClusterManager().GetSession()
	if err != nil {
		return nil, errors.Wrap(err, "get onecloud client session")
	}

	cli := onecloudcli.NewClientSets(s)
	vars, err := d.GetKubesprayVars(cluster)
	if err != nil {
		return nil, errors.Wrap(err, "get variables")
	}

	ms, err := cluster.GetDeployMachines(nil)
	if err != nil {
		return nil, errors.Wrap(err, "get deploy machines")
	}

	hosts, err := d.GetKubesprayInventory(vars, cli, cluster, ms)
	if err != nil {
		return nil, errors.Wrap(err, "get kubespray inventory")
	}
	if len(hosts) == 0 {
		return nil, errors.Error("inventory hosts is empty")
	}

	privateKey := hosts[0].GetPrivateKey()
	consInventoryContent := func(tmpHosts []*kubespray.KubesprayInventoryHost) (string, error) {
		for _, host := range tmpHosts {
			host.Clear()
		}
		return kubespray.NewKubesprayInventory(cluster.GetVersion(), tmpHosts...).ToString()
	}

	content, err := consInventoryContent(hosts)
	if err != nil {
		return nil, errors.Wrap(err, "construct inventory content")
	}

	conf := &api.ClusterKubesprayConfig{
		InventoryContent: content,
		PrivateKey:       privateKey,
		Vars:             jsonutils.Marshal(vars),
	}

	return conf, nil
}

func (d *selfBuildDriver) deployCluster(ctx context.Context, cli onecloudcli.IClient, cluster *models.SCluster, action api.ClusterDeployAction, ms []manager.IMachine) error {
	vars, err := d.GetKubesprayVars(cluster)
	if err != nil {
		return errors.Wrap(err, "get kubespray vars")
	}

	switch action {
	case api.ClusterDeployActionCreate, api.ClusterDeployActionRun:
		err = d.deployClusterByCreate(ctx, cli, cluster, vars, ms)
	case api.ClusterDeployActionScale:
		err = d.deployClusterByScale(ctx, cli, cluster, vars, ms)
	case api.ClusterDeployActionRemoveNode:
		err = d.deployClusterByRemove(ctx, cli, cluster, vars, ms)
	default:
		err = errors.Errorf("Unsupported deploy action: %s", action)
	}

	return err

}

func (d *selfBuildDriver) deployClusterByCreate(
	ctx context.Context,
	cli onecloudcli.IClient,
	cluster *models.SCluster,
	vars *kubespray.KubesprayRunVars,
	ms []manager.IMachine,
) error {
	hosts, err := d.GetKubesprayInventory(vars, cli, cluster, ms)
	if err != nil {
		return errors.Wrap(err, "new kubespray inventory hosts")
	}

	if err := kubespray.NewDefaultKubesprayExecutor().Cluster(vars, hosts...).Run(false); err != nil {
		return errors.Wrap(err, "run kubespray error")
	}

	primaryMaster := hosts[0]
	if err := d.setCAKeyPair(cli, cluster, primaryMaster); err != nil {
		return errors.Wrapf(err, "set cluster ca key pair from %s", primaryMaster.Hostname)
	}

	return nil
}

func (d *selfBuildDriver) deployClusterByAction(
	ctx context.Context,
	cli onecloudcli.IClient,
	cluster *models.SCluster,
	vars *kubespray.KubesprayRunVars,
	targetMs []manager.IMachine,
	actionFunc func(hosts, targetHosts []*kubespray.KubesprayInventoryHost, debug bool) error,
) error {
	allMs, err := cluster.GetDeployMachines(targetMs)
	if err != nil {
		return errors.Wrap(err, "get cluster all deploy machines")
	}
	hosts, err := d.GetKubesprayInventory(vars, cli, cluster, allMs)
	if err != nil {
		return errors.Wrap(err, "new kubespray inventory hosts")
	}

	targetHosts := make([]*kubespray.KubesprayInventoryHost, len(targetMs))

	findInventoryHost := func(hosts []*kubespray.KubesprayInventoryHost, name string) *kubespray.KubesprayInventoryHost {
		for _, h := range hosts {
			if h.Hostname == name {
				return h
			}
		}
		return nil
	}

	for idx := range targetMs {
		name := targetMs[idx].GetName()
		iHost := findInventoryHost(hosts, name)
		if iHost == nil {
			return errors.Errorf("Not found inventory host by name %s", name)
		}
		targetHosts[idx] = iHost
	}

	if err := actionFunc(hosts, targetHosts, false); err != nil {
		return errors.Wrap(err, "run kubespray error")
	}

	return nil
}

func (d *selfBuildDriver) deployClusterByScale(
	ctx context.Context,
	cli onecloudcli.IClient,
	cluster *models.SCluster,
	vars *kubespray.KubesprayRunVars,
	addedMs []manager.IMachine,
) error {
	return d.deployClusterByAction(
		ctx, cli, cluster, vars, addedMs,
		func(hosts, addedHosts []*kubespray.KubesprayInventoryHost, debug bool) error {
			return kubespray.NewDefaultKubesprayExecutor().Scale(vars, hosts, addedHosts...).Run(debug)
		},
	)
}

func (d *selfBuildDriver) deployClusterByRemove(
	ctx context.Context,
	cli onecloudcli.IClient,
	cluster *models.SCluster,
	vars *kubespray.KubesprayRunVars,
	removeMs []manager.IMachine,
) error {
	return d.deployClusterByAction(
		ctx, cli, cluster, vars, removeMs,
		func(hosts, removeHosts []*kubespray.KubesprayInventoryHost, debug bool) error {
			return kubespray.NewDefaultKubesprayExecutor().RemoveNode(vars, hosts, removeHosts...).Run(debug)
		},
	)
}

func (d *selfBuildDriver) getCAKeyPair(cli onecloudcli.IClient, primaryMaster *kubespray.KubesprayInventoryHost) (*api.KeyPair, error) {
	caPath := "/etc/kubernetes/pki/ca.crt"
	caKeyPath := "/etc/kubernetes/pki/ca.key"

	// TODO: check use kubespray host or ip
	host := primaryMaster.AnsibleHost
	user := primaryMaster.User
	passwd := primaryMaster.Password
	key := primaryMaster.GetPrivateKey()

	remoteCat := func(fp string) ([]byte, error) {
		out, err := ssh.RemoteSSHCommand(host, 22, user, passwd, key, fmt.Sprintf("sudo cat %s", fp))
		if err != nil {
			return nil, errors.Wrapf(err, "get %s content", fp)
		}
		return []byte(out), nil
	}

	caOut, err := remoteCat(caPath)
	if err != nil {
		return nil, errors.Wrap(err, "get ca content")
	}
	caKeyOut, err := remoteCat(caKeyPath)
	if err != nil {
		return nil, errors.Wrap(err, "get ca key content")
	}

	return &api.KeyPair{
		Cert: caOut,
		Key:  caKeyOut,
	}, nil
}

func (d *selfBuildDriver) setCAKeyPair(cli onecloudcli.IClient, cluster *models.SCluster, primaryMaster *kubespray.KubesprayInventoryHost) error {
	caKeyPair, err := d.getCAKeyPair(cli, primaryMaster)
	if err != nil {
		return errors.Wrap(err, "get ca key pair")
	}

	if err := cluster.SetCAKeyPair(caKeyPair); err != nil {
		return errors.Wrap(err, "set cluster ca key pair")
	}

	return nil
}

func (d *selfBuildDriver) GetAddonsManifest(cluster *models.SCluster, conf *api.ClusterAddonsManifestConfig) (string, error) {
	commonConf, err := GetCommonAddonsConfig(cluster)
	if err != nil {
		return "", err
	}

	// reg, err := cluster.GetImageRepository()
	// if err != nil {
	// 	return "", err
	// }

	if !cluster.IsInClassicNetwork() {
		commonConf.CloudProviderYunionConfig = nil
		commonConf.IngressControllerYunionConfig = nil
		commonConf.CSIYunionConfig = nil
	}

	pluginConf := &addons.YunionVMPluginsConfig{
		YunionCommonPluginsConfig: commonConf,
		// CNICalicoConfig: &addons.CNICalicoConfig{
		// 	ControllerImage:     registry.MirrorImage(reg.Url, "kube-controllers", "v3.12.1", "calico"),
		// 	NodeImage:           registry.MirrorImage(reg.Url, "node", "v3.12.1", "calico"),
		// 	CNIImage:            registry.MirrorImage(reg.Url, "cni", "v3.12.1", "calico"),
		// 	ClusterCIDR:         cluster.GetPodCidr(),
		// 	EnableNativeIPAlloc: conf.Network.EnableNativeIPAlloc,
		// 	NodeAgentImage:      registry.MirrorImage(reg.Url, "node-agent", "latest", "calico"),
		// },
	}
	return pluginConf.GenerateYAML()
}

func (d *selfBuildDriver) ValidateDeleteMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *models.SCluster, ms []manager.IMachine) error {
	oldMachines, err := cluster.GetMachines()
	if err != nil {
		return err
	}
	for _, m := range ms {
		if len(oldMachines) != len(ms) && m.IsFirstNode() {
			return httperrors.NewInputParameterError("First control node %q must deleted at last", m.GetName())
		}
	}
	return nil
}

func (d *selfBuildDriver) RequestDeleteMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *models.SCluster, ms []manager.IMachine, task taskman.ITask) error {
	items := make([]db.IStandaloneModel, 0)
	for _, m := range ms {
		items = append(items, m.(db.IStandaloneModel))
	}
	return models.MachineManager.StartMachineBatchDeleteTask(ctx, userCred, items, nil, task.GetTaskId())
}

func (d *selfBuildDriver) GetClusterUsers(cluster *models.SCluster, config *rest.Config) ([]api.ClusterUser, error) {
	return nil, nil
}

func (d *selfBuildDriver) GetClusterUserGroups(cluster *models.SCluster, config *rest.Config) ([]api.ClusterUserGroup, error) {
	return nil, nil
}

func (d *selfBuildDriver) ValidateDeleteCondition() error {
	return nil
}

func (d *selfBuildDriver) withKubespray(k8sVersion string, extraConf *api.ClusterExtraConfig) kubespray.KubesprayVars {
	useOffline := false
	provider := d.driver.GetProvider()
	if utils.IsInStringArray(
		string(provider), []string{string(api.ProviderTypeOnecloud), string(api.ProviderTypeOnecloudKvm)}) {
		if options.Options.OfflineRegistryServiceURL != "" || options.Options.OfflineRegistryServiceURL != "" {
			useOffline = true
		}
	}
	if useOffline {
		return kubespray.NewOfflineVars(k8sVersion, extraConf)
	}
	return kubespray.NewDefaultVars(k8sVersion, extraConf)
}
