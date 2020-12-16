package models

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
	"yunion.io/x/kubecomps/pkg/kubeserver/clientv2"
	"yunion.io/x/kubecomps/pkg/kubeserver/constants"
	k8sutil "yunion.io/x/kubecomps/pkg/kubeserver/k8s/util"
	"yunion.io/x/kubecomps/pkg/kubeserver/models/manager"
	"yunion.io/x/kubecomps/pkg/utils/certificates"
	"yunion.io/x/kubecomps/pkg/utils/tokens"
)

var ClusterManager *SClusterManager

func init() {
	initGlobalClusterManager()
}

func GetClusterManager() *SClusterManager {
	if ClusterManager == nil {
		initGlobalClusterManager()
	}
	return ClusterManager
}

func initGlobalClusterManager() {
	if ClusterManager != nil {
		return
	}
	ClusterManager = &SClusterManager{
		SStatusDomainLevelResourceBaseManager: db.NewStatusDomainLevelResourceBaseManager(
			SCluster{},
			"kubeclusters_tbl",
			"kubecluster",
			"kubeclusters",
		),
		SSyncableManager: newSyncableManager(),
	}
	manager.RegisterClusterManager(ClusterManager)
	ClusterManager.SetVirtualObject(ClusterManager)
	ClusterManager.SetAlias("cluster", "clusters")
}

// +onecloud:swagger-gen-model-singular=kubecluster
type SClusterManager struct {
	db.SStatusDomainLevelResourceBaseManager
	SSyncableK8sBaseResourceManager

	*SSyncableManager
}

type SCluster struct {
	db.SStatusDomainLevelResourceBase
	SSyncableK8sBaseResource

	// imported cluster CloudregionId and VpcId is null
	CloudregionId string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" json:"cloudregion_id"`
	VpcId         string `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional" json:"vpc_id"`

	IsSystem bool `nullable:"true" default:"false" list:"admin" create:"optional" json:"is_system"`

	ClusterType     string               `width:"36" charset:"ascii" nullable:"false" create:"required" list:"user"`
	CloudType       string               `width:"36" charset:"ascii" nullable:"false" create:"required" list:"user"`
	ResourceType    string               `width:"36" charset:"ascii" nullable:"false" create:"required" list:"user"`
	Mode            string               `width:"36" charset:"ascii" nullable:"false" create:"required" list:"user"`
	Provider        string               `width:"36" charset:"ascii" nullable:"false" create:"required" list:"user"`
	ServiceCidr     string               `width:"36" charset:"ascii" nullable:"false" create:"required" list:"user"`
	ServiceDomain   string               `width:"128" charset:"ascii" nullable:"false" create:"required" list:"user"`
	PodCidr         string               `width:"36" charset:"ascii" nullable:"true" create:"optional" list:"user"`
	Ha              tristate.TriState    `nullable:"true" create:"required" list:"user"`
	ImageRepository jsonutils.JSONObject `nullable:"true" create:"optional" list:"user"`

	// kubernetes config
	Kubeconfig string `nullable:"true" charset:"utf8" create:"optional"`

	// kubernetes api server endpoint
	ApiServer string `width:"256" nullable:"true" charset:"ascii" create:"optional" list:"user"`

	// Version records kubernetes api server version
	Version string `width:"128" charset:"ascii" nullable:"false" create:"optional" list:"user"`
	// kubernetes distribution
	Distribution string `width:"256" nullable:"true" default:"k8s" charset:"utf8" create:"optional" list:"user"`
	// DistributionInfo records distribution misc info
	DistributionInfo jsonutils.JSONObject `nullable:"true" create:"optional" list:"user"`
	// AddonsConfig records cluster addons config
	AddonsConfig jsonutils.JSONObject `nullable:"true" create:"optional" list:"user"`
}

func (m *SClusterManager) InitializeData() error {
	clusters := []SCluster{}
	q := m.Query().IsNullOrEmpty("resource_type")
	err := db.FetchModelObjects(m, q, &clusters)
	if err != nil {
		return err
	}
	for _, cluster := range clusters {
		tmp := &cluster
		db.Update(tmp, func() error {
			tmp.ResourceType = string(api.ClusterResourceTypeHost)
			return nil
		})
	}
	return nil
}

func (m *SClusterManager) SyncClustersFromCloud(ctx context.Context) error {
	clusters, err := m.GetClusters()
	if err != nil {
		return errors.Wrap(err, "get all clusters")
	}

	s, err := m.GetSession()
	if err != nil {
		return errors.Wrap(err, "get auth session")
	}

	var errs []error
	for _, cls := range clusters {
		if err := cls.SyncInfoFromCloud(ctx, s); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.NewAggregate(errs)
}

func (c *SCluster) LogPrefix() string {
	return c.GetId() + "/" + c.GetName()
}

func (c *SCluster) GetMode() api.ModeType {
	return api.ModeType(c.Mode)
}

func (c *SCluster) IsImported() bool {
	if c.GetMode() == api.ModeTypeImport {
		return true
	}
	return false
}

func (c *SCluster) SyncInfoFromCloud(ctx context.Context, s *mcclient.ClientSession) error {
	// imported cluster not need sync info from cloud
	if c.IsImported() {
		return nil
	}

	// for classic network cluster
	_, err := db.Update(c, func() error {
		if c.CloudregionId == "" {
			c.CloudregionId = computeapi.DEFAULT_REGION_ID
			if c.VpcId == "" {
				c.VpcId = computeapi.DEFAULT_VPC_ID
			}
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "update cluster %s info", c.GetName())
	}

	ms, err := c.GetMachines()
	if err != nil {
		return errors.Wrap(err, "get cluster machines")
	}
	var errs []error
	for _, m := range ms {
		if m.GetResourceId() == "" {
			continue
		}
		if err := m.SyncInfoFromCloud(ctx, s); err != nil {
			errs = append(errs, errors.Wrapf(err, "sync cluster %s machine from cloud", c.LogPrefix()))
		}
	}
	return errors.NewAggregate(errs)
}

func (m *SClusterManager) FilterByHiddenSystemAttributes(q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	q = m.SStatusDomainLevelResourceBaseManager.FilterByHiddenSystemAttributes(q, userCred, query, scope)
	isSystem := jsonutils.QueryBoolean(query, "system", false)
	if isSystem {
		var isAllow bool
		if consts.IsRbacEnabled() {
			allowScope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), m.KeywordPlural(), policy.PolicyActionList, "system")
			if !scope.HigherThan(allowScope) {
				isAllow = true
			}
		} else {
			if userCred.HasSystemAdminPrivilege() {
				isAllow = true
			}
		}
		if !isAllow {
			isSystem = false
		}
	}
	if !isSystem {
		q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(q.Field("is_system")), sqlchemy.IsFalse(q.Field("is_system"))))
	}
	return q
}

func (m *SClusterManager) GetSystemCluster() (*SCluster, error) {
	clusters := m.Query().SubQuery()
	q := clusters.Query().Filter(sqlchemy.Equals(clusters.Field("provider"), string(api.ProviderTypeSystem)))
	q = q.Equals("name", SystemClusterName)
	objs := make([]SCluster, 0)
	err := db.FetchModelObjects(m, q, &objs)
	if err != nil {
		return nil, err
	}
	if len(objs) == 0 {
		// return nil, httperrors.NewNotFoundError("Not found default system cluster")
		return nil, nil
	}
	if len(objs) != 1 {
		return nil, httperrors.NewDuplicateResourceError("Found %d system cluster", len(objs))
	}
	sysCluster := objs[0]
	return &sysCluster, nil
}

func (m *SClusterManager) GetSystemClusterConfig() (*rest.Config, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

const (
	SystemClusterName = "system-default"
	NamespaceOneCloud = "onecloud"
)

type k8sInfo struct {
	ApiServer  string
	Kubeconfig string
}

func (m *SClusterManager) GetSystemClusterK8SInfo() (*k8sInfo, error) {
	restCfg, err := m.GetSystemClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "get rest config")
	}
	kubeconfig, err := m.GetSystemClusterKubeconfig(restCfg.Host, restCfg)
	if err != nil {
		return nil, errors.Wrap(err, "generate k8s kubeconfig")
	}
	return &k8sInfo{
		ApiServer:  restCfg.Host,
		Kubeconfig: kubeconfig,
	}, nil
}

func (m *SClusterManager) GetSystemClusterCreateData() (*api.ClusterCreateInput, error) {
	createData := &api.ClusterCreateInput{
		ClusterType: api.ClusterTypeDefault,
		CloudType:   api.CloudTypePrivate,
		Mode:        api.ModeTypeImport,
		Provider:    api.ProviderTypeSystem,
	}
	createData.Name = SystemClusterName
	k8sInfo, err := m.GetSystemClusterK8SInfo()
	if err != nil {
		return nil, errors.Wrap(err, "get k8s info")
	}
	importData := &api.ImportClusterData{
		ApiServer:  k8sInfo.ApiServer,
		Kubeconfig: k8sInfo.Kubeconfig,
	}
	createData.ImportData = importData
	return createData, nil
}

func (m *SClusterManager) RegisterSystemCluster() error {
	sysCluster, err := m.GetSystemCluster()
	if err != nil {
		return errors.Wrap(err, "get system cluster")
	}
	userCred := GetAdminCred()
	newCreated := false
	ctx := context.TODO()
	if sysCluster == nil {
		// create system cluster
		createData, err := m.GetSystemClusterCreateData()
		if err != nil {
			return errors.Wrap(err, "get cluster create data")
		}
		obj, err := db.DoCreate(m, context.TODO(), userCred, nil, createData.JSON(createData), userCred)
		if err != nil {
			return errors.Wrap(err, "create cluster")
		}
		func() {
			lockman.LockObject(ctx, obj)
			defer lockman.ReleaseObject(ctx, obj)

			obj.PostCreate(ctx, userCred, userCred, nil, createData.JSON(createData))
		}()
		sysCluster = obj.(*SCluster)
		newCreated = true
	}
	// update system cluster
	k8sInfo, err := m.GetSystemClusterK8SInfo()
	if err != nil {
		return errors.Wrap(err, "get k8s info")
	}
	if _, err := db.Update(sysCluster, func() error {
		if sysCluster.ApiServer != k8sInfo.ApiServer {
			sysCluster.ApiServer = k8sInfo.ApiServer
		}
		if sysCluster.Kubeconfig != k8sInfo.Kubeconfig {
			sysCluster.Kubeconfig = k8sInfo.Kubeconfig
		}
		if !sysCluster.IsSystem {
			sysCluster.IsSystem = true
		}
		return nil
	}); err != nil {
		return errors.Wrap(err, "update system cluster")
	}
	if !newCreated {
		if err := sysCluster.StartSyncStatus(ctx, userCred, ""); err != nil {
			return errors.Wrap(err, "start sysCluster sync status task")
		}
	}
	for i := 0; i < 5; i++ {
		sysCluster, err = GetClusterManager().GetSystemCluster()
		if err != nil {
			return errors.Wrap(err, "get system cluster")
		}
		if sysCluster.GetStatus() != api.ClusterStatusRunning {
			log.Warningf("system cluster status %s != running", sysCluster.GetStatus())
			time.Sleep(5 * time.Second)
			continue
		}
		return nil
	}
	return errors.Errorf("system cluster status %s not running", sysCluster.GetStatus())
}

func SetJSONDataDefault(data *jsonutils.JSONDict, key string, defVal string) string {
	val, _ := data.GetString(key)
	if len(val) == 0 {
		val = defVal
		data.Set(key, jsonutils.NewString(val))
	}
	return val
}

func (m *SClusterManager) GetSession() (*mcclient.ClientSession, error) {
	return GetAdminSession()
}

func (m *SClusterManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.ClusterListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SStatusDomainLevelResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StatusDomainLevelResourceListInput)
	if err != nil {
		return nil, err
	}
	if input.FederatedResourceUsedInput.ShouldDo() {
		fedJointMan := GetFedJointClusterManager(input.FederatedKeyword)
		if fedJointMan == nil {
			return nil, httperrors.NewInputParameterError("federated_keyword %s not found", input.FederatedKeyword)
		}
		fedMan := fedJointMan.GetFedManager()
		fedObj, err := fedMan.FetchByIdOrName(userCred, input.FederatedResourceId)
		if err != nil {
			return nil, httperrors.NewNotFoundError("federated resource %s %s found error: %v", input.FederatedKeyword, input.FederatedResourceId, err)
		}
		sq := fedJointMan.Query("cluster_id").Equals("federatedresource_id", fedObj.GetId()).SubQuery()
		if *input.FederatedUsed {
			q = q.In("id", sq)
		} else {
			q = q.NotIn("id", sq)
		}
	}
	return q, nil
}

func (m *SClusterManager) CreateCluster(ctx context.Context, userCred mcclient.TokenCredential, data api.ClusterCreateInput) (manager.ICluster, error) {
	input := jsonutils.Marshal(data)
	obj, err := db.DoCreate(m, ctx, userCred, nil, input, userCred)
	if err != nil {
		return nil, err
	}
	return obj.(*SCluster), nil
}

func (m *SClusterManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.ClusterCreateInput) (*api.ClusterCreateInput, error) {
	sInput, err := m.SStatusDomainLevelResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StatusDomainLevelResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.StatusDomainLevelResourceCreateInput = sInput
	if input.IsSystem != nil && *input.IsSystem && !db.IsAdminAllowCreate(userCred, m) {
		return nil, httperrors.NewNotSufficientPrivilegeError("non-admin user not allowed to create system object")
	}

	if input.ClusterType == "" {
		input.ClusterType = api.ClusterTypeDefault
	}
	if !utils.IsInStringArray(string(input.ClusterType), []string{string(api.ClusterTypeDefault)}) {
		return nil, httperrors.NewInputParameterError("Invalid cluster type: %q", input.ClusterType)
	}

	if input.CloudType == "" {
		input.CloudType = api.CloudTypePrivate
	}
	if !utils.IsInStringArray(string(input.CloudType), []string{string(api.CloudTypePrivate)}) {
		return nil, httperrors.NewInputParameterError("Invalid cloud type: %q", input.CloudType)
	}

	if input.ResourceType == "" {
		input.ResourceType = api.ClusterResourceTypeHost
	}
	if err := m.ValidateResourceType(string(input.ResourceType)); err != nil {
		return nil, err
	}

	if input.Mode == "" {
		input.Mode = api.ModeTypeSelfBuild
	}
	if !utils.IsInStringArray(string(input.Mode), []string{
		string(api.ModeTypeSelfBuild),
		string(api.ModeTypeImport),
	}) {
		return nil, httperrors.NewInputParameterError("Invalid mode type: %q", input.Mode)
	}

	if input.Provider == "" {
		input.Provider = api.ProviderTypeOnecloud
	}
	if err := m.ValidateProviderType(string(input.Provider)); err != nil {
		return nil, err
	}

	driver, err := GetDriverWithError(
		input.Mode,
		input.Provider,
		input.ResourceType,
	)
	if err != nil {
		return nil, err
	}

	// TODO: fetch serviceCidr, serviceDomain and podCidr from import cluster
	if input.ServiceCidr == "" {
		input.ServiceCidr = api.DefaultServiceCIDR
	}
	if _, err := netutils.NewIPV4Prefix(input.ServiceCidr); err != nil {
		return nil, httperrors.NewInputParameterError("Invalid service CIDR: %q", input.ServiceCidr)
	}
	if input.ServiceDomain == "" {
		input.ServiceDomain = api.DefaultServiceDomain
	}
	if len(input.ServiceDomain) == 0 {
		return nil, httperrors.NewInputParameterError("service domain must provided")
	}
	if input.PodCidr == "" {
		input.PodCidr = api.DefaultPodCIDR
	}
	if _, err := netutils.NewIPV4Prefix(input.PodCidr); err != nil {
		return nil, httperrors.NewInputParameterError("Invalid pod CIDR: %q", input.PodCidr)
	}

	if input.Provider != api.ProviderTypeSystem && driver.NeedCreateMachines() && len(input.Machines) == 0 {
		return nil, httperrors.NewInputParameterError("Machines desc not provider")
	}

	var machineResType api.MachineResourceType
	for _, m := range input.Machines {
		if len(m.ResourceType) == 0 {
			return nil, httperrors.NewInputParameterError("Machine resource type is empty")
		}
		if len(machineResType) == 0 {
			machineResType = api.MachineResourceType(m.ResourceType)
		}
		if string(machineResType) != m.ResourceType {
			return nil, httperrors.NewInputParameterError("Machine resource type must same")
		}
	}

	if err := driver.ValidateCreateData(ctx, userCred, ownerId, query, input); err != nil {
		return nil, err
	}

	versions := driver.GetK8sVersions()
	if len(versions) > 0 {
		defaultVersion := versions[0]
		if input.Version == "" {
			input.Version = defaultVersion
		}
		if !utils.IsInStringArray(input.Version, versions) {
			return nil, httperrors.NewInputParameterError("Invalid version: %q, choose one from %v", input.Version, versions)
		}
	}

	imageRepo := input.ImageRepository
	if imageRepo != nil {
		if imageRepo.Url == "" {
			return nil, httperrors.NewNotEmptyError("image_repository.url is empty, use format: 'registry.hub.docker.com/yunion'")
		}
		if _, err := m.GetRegistryUrlByRepoUrl(imageRepo.Url); err != nil {
			return nil, err
		}
	}
	return input, nil
}

func (cluster *SCluster) GetDistributionInfo() (*api.ClusterDistributionInfo, error) {
	out := new(api.ClusterDistributionInfo)
	if err := cluster.DistributionInfo.Unmarshal(out); err != nil {
		return nil, err
	}
	return out, nil
}

func (cluster *SCluster) GetAddonsConfig() (*api.ClusterAddonsManifestConfig, error) {
	out := new(api.ClusterAddonsManifestConfig)
	if err := cluster.AddonsConfig.Unmarshal(out); err != nil {
		return nil, err
	}
	return out, nil
}

func (cluster *SCluster) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if err := cluster.SStatusDomainLevelResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data); err != nil {
		return err
	}
	input := new(api.ClusterCreateInput)
	if err := data.Unmarshal(input); err != nil {
		return errors.Wrap(err, "unmarshal cluster create input")
	}
	if input.IsSystem != nil && *input.IsSystem {
		cluster.IsSystem = true
	} else {
		cluster.IsSystem = false
	}
	if input.Mode == api.ModeTypeImport {
		cluster.Kubeconfig = input.ImportData.Kubeconfig
		cluster.Distribution = input.ImportData.Distribution
		cluster.ApiServer = input.ImportData.ApiServer
		cluster.DistributionInfo = jsonutils.Marshal(input.ImportData.DistributionInfo)
	}
	return nil
}

func (m *SClusterManager) AllowGetPropertyK8sVersions(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (m *SClusterManager) ValidateProviderType(providerType string) error {
	if !utils.IsInStringArray(providerType, []string{
		string(api.ProviderTypeOnecloud),
		string(api.ProviderTypeSystem),
		string(api.ProviderTypeExternal),
	}) {
		return httperrors.NewInputParameterError("Invalid provider type: %q", providerType)
	}
	return nil
}

func (m *SClusterManager) ValidateResourceType(resType string) error {
	if !utils.IsInStringArray(resType, []string{
		string(api.ClusterResourceTypeHost),
		string(api.ClusterResourceTypeGuest),
		string(api.ClusterResourceTypeUnknown),
	}) {
		return httperrors.NewInputParameterError("Invalid cluster resource type: %q", resType)
	}
	return nil
}

func (m *SClusterManager) GetDriverByQuery(query jsonutils.JSONObject) (IClusterDriver, error) {
	modeType, _ := query.GetString("mode")
	providerType, _ := query.GetString("provider")
	resType, _ := query.GetString("resource_type")
	if err := m.ValidateProviderType(providerType); err != nil {
		return nil, err
	}
	if len(resType) == 0 {
		resType = string(api.ClusterResourceTypeHost)
	}
	if err := m.ValidateResourceType(resType); err != nil {
		return nil, err
	}
	driver := GetClusterDriver(
		api.ModeType(modeType),
		api.ProviderType(providerType),
		api.ClusterResourceType(resType))
	return driver, nil
}

func (m *SClusterManager) GetPropertyK8sVersions(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	SetJSONDataDefault(query.(*jsonutils.JSONDict), "mode", string(api.ModeTypeSelfBuild))
	driver, err := m.GetDriverByQuery(query)
	if err != nil {
		return nil, err
	}
	versions := driver.GetK8sVersions()
	ret := jsonutils.Marshal(versions)
	return ret, nil
}

func (m *SClusterManager) AllowPerformCheckSystemReady(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return true
}

func (m *SClusterManager) PerformCheckSystemReady(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	isReady, err := m.IsSystemClusterReady()
	if err != nil {
		return nil, err
	}
	return jsonutils.Marshal(isReady), nil
}

func (m *SClusterManager) IsSystemClusterReady() (bool, error) {
	clusters := m.Query().SubQuery()
	q := clusters.Query()
	q = q.Filter(sqlchemy.Equals(clusters.Field("status"), api.ClusterStatusRunning))
	cnt, err := q.CountWithError()
	if err != nil {
		return false, err
	}
	if cnt <= 0 {
		return false, nil
	}
	return true, nil
}

func (m *SClusterManager) AllowGetPropertyUsableInstances(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return userCred.IsAllow(rbacutils.ScopeSystem, m.KeywordPlural(), policy.PolicyActionGet, "usable-instances")
}

func (m *SClusterManager) GetPropertyUsableInstances(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	SetJSONDataDefault(query.(*jsonutils.JSONDict), "mode", string(api.ModeTypeSelfBuild))
	driver, err := m.GetDriverByQuery(query)
	if err != nil {
		return nil, err
	}
	session, err := m.GetSession()
	if err != nil {
		return nil, err
	}
	instances, err := driver.GetUsableInstances(session)
	if err != nil {
		return nil, err
	}
	ret := jsonutils.Marshal(instances)
	return ret, nil
}

// PerformGC cleanup clusters related orphan resources
func (m *SClusterManager) PerformGc(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	go func() {
		subMans := m.GetSubManagers()
		for _, man := range subMans {
			if err := GetClusterResAPI().PerformGC(man, ctx, userCred); err != nil {
				log.Errorf("PerformGC %s %v", man.KeywordPlural(), err)
			}
		}
	}()
	return nil, nil
}

func (m *SClusterManager) IsClusterExists(userCred mcclient.TokenCredential, id string) (manager.ICluster, bool, error) {
	obj, err := m.FetchByIdOrName(userCred, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}
	return obj.(*SCluster), true, nil
}

/*func (m *SClusterManager) GetNonSystemClusters() ([]manager.ICluster, error) {
	clusters := m.Query().SubQuery()
	q := clusters.Query().Filter(sqlchemy.NotEquals(clusters.Field("provider"), string(types.ProviderTypeSystem)))
	objs := make([]SCluster, 0)
	err := db.FetchModelObjects(m, q, &objs)
	if err != nil {
		return nil, err
	}
	ret := make([]manager.ICluster, len(objs))
	for i := range objs {
		ret[i] = &objs[i]
	}
	return ret, nil
}*/

func (m *SClusterManager) GetRunningClusters() ([]manager.ICluster, error) {
	return m.GetClustersByStatus(api.ClusterStatusRunning)
}

func (m *SClusterManager) GetClusters() ([]manager.ICluster, error) {
	return m.GetClustersByStatus()
}

func (m *SClusterManager) getClustersByStatus(status ...string) ([]SCluster, error) {
	q := m.Query()
	if len(status) != 0 {
		q = q.In("status", status)
	}
	objs := make([]SCluster, 0)
	err := db.FetchModelObjects(m, q, &objs)
	if err != nil {
		return nil, err
	}
	return objs, nil
}

func (m *SClusterManager) GetClustersByStatus(status ...string) ([]manager.ICluster, error) {
	objs, err := m.getClustersByStatus(status...)
	if err != nil {
		return nil, err
	}
	ret := make([]manager.ICluster, len(objs))
	for i := range objs {
		ret[i] = &objs[i]
	}
	return ret, nil
}

func (m *SClusterManager) FetchClusterByIdOrName(userCred mcclient.TokenCredential, id string) (manager.ICluster, error) {
	return m.GetClusterByIdOrName(userCred, id)
}

func (m *SClusterManager) GetClusterByIdOrName(userCred mcclient.TokenCredential, id string) (*SCluster, error) {
	cluster, err := m.FetchByIdOrName(userCred, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, httperrors.NewNotFoundError("Cluster %s", id)
		}
		return nil, err
	}
	return cluster.(*SCluster), nil
}

func (m *SClusterManager) GetCluster(id string) (*SCluster, error) {
	obj, err := m.FetchById(id)
	if err != nil {
		return nil, err
	}
	return obj.(*SCluster), nil
}

func (m *SClusterManager) ClusterHealthCheckTask(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	clusters, err := m.getClustersByStatus(
		api.ClusterStatusRunning,
		api.ClusterStatusLost,
		//types.ClusterStatusUnknown,
	)
	if err != nil {
		log.Errorf("ClusterHealthCheckTask get clusters: %v", err)
		return
	}
	for idx := range clusters {
		c := &clusters[idx]
		if err := c.IsHealthy(); err == nil {
			prevStatus := c.GetStatus()
			if c.GetStatus() != api.ClusterStatusRunning {
				if err := c.SetStatus(userCred, api.ClusterStatusRunning, "by health check cronjob"); err != nil {
					log.Errorf("Set cluster %s status to running error: %v", c.GetName(), err)
				} else {
					c.Status = api.ClusterStatusRunning
					log.Errorf("===set cluster %s status succ: %s", c.GetName(), c.GetStatus())
				}
				if err := client.GetClustersManager().UpdateClient(c); err != nil {
					log.Errorf("Update cluster %s client error: %v", c.GetName(), err)
					c.SetStatus(userCred, prevStatus, err.Error())
				} else {
					if err := c.StartSyncTask(ctx, userCred, nil, ""); err != nil {
						log.Errorf("cluster %s StartSyncTask when health check error: %v", c.GetId(), err)
					}
				}
			}
			continue
		} else {
			c.SetStatus(userCred, api.ClusterStatusLost, err.Error())
			client.GetClustersManager().RemoveClient(c.GetId())
		}
	}
}

func (c *SCluster) GetDriver() IClusterDriver {
	return GetClusterDriver(
		api.ModeType(c.Mode),
		api.ProviderType(c.Provider),
		api.ClusterResourceType(c.ResourceType))
}

func (c *SCluster) GetMachinesCount() (int, error) {
	ms, err := c.GetMachines()
	if err != nil {
		return 0, err
	}
	return len(ms), nil
}

func (c *SCluster) GetNodesCount() (int, error) {
	cli, err := c.GetRemoteClient()
	if err != nil {
		return 0, errors.Wrap(err, "get cluster client")
	}
	lister := cli.GetHandler().GetIndexer().NodeLister()
	nodes, err := lister.List(labels.Everything())
	if err != nil {
		return 0, errors.Wrap(err, "list k8s nodes")
	}
	return len(nodes), nil
}

func (man *SClusterManager) GetImageRepository(input *api.ImageRepository) *api.ImageRepository {
	ret := &api.ImageRepository{
		Url: constants.DefaultRegistryMirror,
	}
	if input == nil {
		return ret
	}
	if input.Url != "" {
		ret.Url = input.Url
	}
	ret.Insecure = input.Insecure
	return ret
}

func (c *SCluster) GetImageRepository() (*api.ImageRepository, error) {
	ret := new(api.ImageRepository)
	if c.ImageRepository == nil {
		return ClusterManager.GetImageRepository(nil), nil
	}
	if err := c.ImageRepository.Unmarshal(ret); err != nil {
		return nil, err
	}
	return ClusterManager.GetImageRepository(ret), nil
}

func (c *SCluster) IsHealthy() error {
	cli, err := c.GetK8sClient()
	if err != nil {
		return err
	}
	if _, err := cli.Discovery().ServerVersion(); err != nil {
		return err
	}
	return nil
}

func (m *SClusterManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []*jsonutils.JSONDict {
	rows := make([]*jsonutils.JSONDict, len(objs))
	virtRows := m.SStatusDomainLevelResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range objs {
		rows[i] = jsonutils.Marshal(virtRows[i]).(*jsonutils.JSONDict)
		rows[i] = objs[i].(*SCluster).moreExtraInfo(rows[i])
	}
	return rows
}

func (c *SCluster) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (*jsonutils.JSONDict, error) {
	extra, err := c.SStatusDomainLevelResourceBase.GetExtraDetails(ctx, userCred, query, isList)
	if err != nil {
		return nil, err
	}

	return c.moreExtraInfo(jsonutils.Marshal(extra).(*jsonutils.JSONDict)), nil
}

func (c *SCluster) moreExtraInfo(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	var cnt int
	var err error
	cnt, _ = c.GetMachinesCount()
	if cnt == 0 {
		cnt, err = c.GetNodesCount()
	}
	if err != nil {
		log.Errorf("get machines count error: %v", err)
	} else {
		extra.Add(jsonutils.NewInt(int64(cnt)), "machines")
	}
	return extra
}

type CertificatesGroup struct {
	CAKeyPair           *SX509KeyPair
	EtcdCAKeyPair       *SX509KeyPair
	FrontProxyCAKeyPair *SX509KeyPair
	SAKeyPair           *SX509KeyPair
}

func (c *SCluster) GetCertificatesGroup() (*CertificatesGroup, error) {
	caKp, err := c.GetCAKeyPair()
	if err != nil {
		return nil, errors.Wrap(err, "get CAKeyPair")
	}
	etcdKp, err := c.GetEtcdCAKeyPair()
	if err != nil {
		return nil, errors.Wrap(err, "get EtcdCAKeyPair")
	}
	fpKp, err := c.GetFrontProxyCAKeyPair()
	if err != nil {
		return nil, errors.Wrap(err, "get FrontProxyCAKeyPair")
	}
	saKp, err := c.GetSAKeyPair()
	if err != nil {
		return nil, errors.Wrap(err, "get ServiceAccount KeyPair")
	}
	return &CertificatesGroup{
		CAKeyPair:           caKp,
		EtcdCAKeyPair:       etcdKp,
		FrontProxyCAKeyPair: fpKp,
		SAKeyPair:           saKp,
	}, nil
}

func (man *SClusterManager) GetRegistryUrlByRepoUrl(imageRepo string) (string, error) {
	rets := strings.Split(imageRepo, "/")
	if len(rets) != 2 {
		return "", httperrors.NewInputParameterError("Invalid image repository format %s, use like: 'docker.io/yunion'", imageRepo)
	}
	return rets[0], nil
}

func (c *SCluster) GetDefaultMachineDockerConfig(imageRepo *api.ImageRepository) (*api.DockerConfig, error) {
	ret := new(api.DockerConfig)
	if imageRepo.Insecure {
		reg, err := ClusterManager.GetRegistryUrlByRepoUrl(imageRepo.Url)
		if err != nil {
			return nil, err
		}
		ret.InsecureRegistries = []string{reg}
	}
	return ret, nil
}

func (c *SCluster) FillMachinePrepareInput(input *api.MachinePrepareInput) (*api.MachinePrepareInput, error) {
	cg, err := c.GetCertificatesGroup()
	if err != nil {
		return nil, errors.Wrap(err, "get certificates group")
	}
	input.CAKeyPair = cg.CAKeyPair.ToKeyPair()
	input.EtcdCAKeyPair = cg.EtcdCAKeyPair.ToKeyPair()
	input.FrontProxyCAKeyPair = cg.FrontProxyCAKeyPair.ToKeyPair()
	input.SAKeyPair = cg.SAKeyPair.ToKeyPair()
	if !input.FirstNode {
		bootstrapToken, err := c.GetNodeJoinToken()
		if err != nil {
			return nil, errors.Wrapf(err, "get %s node join token", input.Role)
		}
		input.BootstrapToken = bootstrapToken
	}
	imageRepo, err := c.GetImageRepository()
	if err != nil {
		return nil, err
	}
	input.Config.ImageRepository = imageRepo
	dockerConfig, err := c.GetDefaultMachineDockerConfig(imageRepo)
	if err != nil {
		return nil, err
	}
	input.Config.DockerConfig = dockerConfig
	// TODO: support lb
	return input, nil
}

func (c *SCluster) GetNodeJoinToken() (string, error) {
	kubeConfig, err := c.GetKubeconfig()
	if err != nil {
		return "", errors.Wrapf(err, "failed to retrieve kubeconfig for cluster %q", c.GetName())
	}
	controlPlaneURL, err := c.GetControlPlaneUrl()
	if err != nil {
		return "", errors.Wrapf(err, "failed to get controlPlaneURL")
	}
	clientConfig, err := clientcmd.BuildConfigFromKubeconfigGetter(controlPlaneURL, func() (*clientcmdapi.Config, error) {
		return clientcmd.Load([]byte(kubeConfig))
	})
	if err != nil {
		return "", errors.Wrapf(err, "failed to get client config for cluster at %q", controlPlaneURL)
	}

	coreClient, err := corev1.NewForConfig(clientConfig)
	if err != nil {
		return "", errors.Wrapf(err, "failed to initialize new corev1 client")
	}

	bootstrapToken, err := tokens.NewBootstrap(coreClient, 24*time.Hour)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create new bootstrap token")
	}
	return bootstrapToken, nil
}

func (c *SCluster) AttachKeypair(ctx context.Context, userCred mcclient.TokenCredential, keypair *SX509KeyPair) error {
	attached, err := c.IsAttachKeypair(keypair)
	if err != nil {
		return errors.Wrapf(err, "check keypair %s attached to cluster %s", keypair.GetName(), c.GetName())
	}
	if attached {
		return errors.Errorf("Cluster %s already attached keypair %s", c.GetName(), keypair.GetName())
	}
	model, err := db.NewModelObject(ClusterX509KeyPairManager)
	if err != nil {
		return errors.Wrapf(err, "new cluster %s keypair %s obj", c.GetName(), keypair.GetName())
	}

	clusterKeypair := model.(*SClusterX509KeyPair)
	clusterKeypair.ClusterId = c.GetId()
	clusterKeypair.KeypairId = keypair.GetId()
	clusterKeypair.User = keypair.User
	return ClusterX509KeyPairManager.TableSpec().Insert(ctx, clusterKeypair)
}

func (c *SCluster) IsAttachKeypair(kp *SX509KeyPair) (bool, error) {
	q := ClusterX509KeyPairManager.Query().Equals("keypair_id", kp.GetId()).Equals("cluster_id", c.GetId())
	cnt, err := q.CountWithError()
	if err != nil {
		return false, err
	}
	return cnt > 0, nil
}

func (c *SCluster) GenerateCertificates(ctx context.Context, userCred mcclient.TokenCredential) error {
	if !c.GetDriver().NeedGenerateCertificate() {
		return nil
	}
	clusterCAKeyPair, err := X509KeyPairManager.GenerateCertificates(ctx, userCred, c, api.ClusterCA)
	if err != nil {
		return errors.Wrapf(err, "Generate %s certificate", api.ClusterCA)
	}
	infof := func(kp *SX509KeyPair) {
		log.Infof("Generate cluster %s %s certificate", c.GetName(), kp.GetName())
	}
	infof(clusterCAKeyPair)
	etcdCAKeyPair, err := X509KeyPairManager.GenerateCertificates(ctx, userCred, c, api.EtcdCA)
	if err != nil {
		return errors.Wrapf(err, "Generate %s certificate", api.EtcdCA)
	}
	infof(etcdCAKeyPair)
	fpCAKeyPair, err := X509KeyPairManager.GenerateCertificates(ctx, userCred, c, api.FrontProxyCA)
	if err != nil {
		return errors.Wrapf(err, "Generate %s certificate", api.FrontProxyCA)
	}
	infof(fpCAKeyPair)
	saKeyPair, err := X509KeyPairManager.GenerateServiceAccountKeys(ctx, userCred, c, api.ServiceAccountCA)
	if err != nil {
		return errors.Wrapf(err, "Generate ServiceAccount key %s", api.ServiceAccountCA)
	}
	infof(saKeyPair)
	return nil
}

func (c *SCluster) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	c.SStatusDomainLevelResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	if err := c.StartClusterCreateTask(ctx, userCred, data.(*jsonutils.JSONDict), ""); err != nil {
		log.Errorf("StartClusterCreateTask error: %v", err)
	}
}

func (c *SCluster) StartClusterCreateTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	c.SetStatus(userCred, api.ClusterStatusCreating, "")
	task, err := taskman.TaskManager.NewTask(ctx, "ClusterCreateTask", c, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (c *SCluster) GetPVCCount() (int, error) {
	cli, err := c.GetK8sClient()
	if err != nil {
		return 0, err
	}
	pvcs, err := k8sutil.GetPVCList(cli, "")
	if err != nil {
		return 0, err
	}
	return len(pvcs.Items), nil
}

func (c *SCluster) CheckPVCEmpty() error {
	pvcCnt, _ := c.GetPVCCount()
	if pvcCnt > 0 {
		return httperrors.NewNotAcceptableError("Cluster has %d PersistentVolumeClaims, clean them firstly", pvcCnt)
	}
	return nil
}

func (c *SCluster) ValidateDeleteCondition(ctx context.Context) error {
	if err := c.GetDriver().ValidateDeleteCondition(); err != nil {
		return err
	}
	return nil
}

func (c *SCluster) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("Cluster delete do nothing")
	return nil
}

func (c *SCluster) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	if err := X509KeyPairManager.DeleteKeyPairsByCluster(ctx, userCred, c); err != nil {
		return errors.Wrapf(err, "DeleteKeyPairsByCluster")
	}
	if err := c.DeleteAllComponents(ctx, userCred); err != nil {
		return errors.Wrapf(err, "DeleteClusterComponent")
	}
	/*
	 * if err := client.GetClustersManager().RemoveClient(c.GetId()); err != nil {
	 *     return errors.Wrap(err, "Delete from client")
	 * }
	 */
	if err := c.PurgeAllClusterResource(ctx, userCred); err != nil {
		return errors.Wrap(err, "Purge all k8s cluster db resources")
	}
	if err := c.PurgeAllFedResource(ctx, userCred); err != nil {
		return errors.Wrap(err, "Purge all federated cluster resources")
	}
	return c.SStatusDomainLevelResourceBase.Delete(ctx, userCred)
}

func (c *SCluster) PurgeAllClusterResource(ctx context.Context, userCred mcclient.TokenCredential) error {
	return c.purgeSubClusterResource(ctx, userCred, GetClusterManager().GetSubManagers())
}

func (c *SCluster) purgeSubClusterResource(ctx context.Context, userCred mcclient.TokenCredential, resMans []ISyncableManager) error {
	if len(resMans) == 0 {
		return nil
	}
	for _, resMan := range resMans {
		log.Infof("Start purge cluster %s(%s) resources %s", c.GetName(), c.GetId(), resMan.KeywordPlural())
		if err := c.purgeSubClusterResource(ctx, userCred, resMan.GetSubManagers()); err != nil {
			return errors.Wrapf(err, "purge resource %s subresource", resMan.KeywordPlural())
		}
		if err := resMan.PurgeAllByCluster(ctx, userCred, c); err != nil {
			return errors.Wrapf(err, "purge subresource %s", resMan.Keyword())
		}
		log.Infof("Purge cluster %s resources %s success.", c.GetName(), resMan.KeywordPlural())
	}
	return nil
}

func (c *SCluster) PurgeAllFedResource(ctx context.Context, userCred mcclient.TokenCredential) error {
	for _, m := range GetFedManagers() {
		log.Infof("start purge federated %s joint resource for cluster", m.KeywordPlural(), c.GetName())
		if err := m.PurgeAllByCluster(ctx, userCred, c); err != nil {
			return errors.Wrapf(err, "purge federated resource %s for cluster %s", m.Keyword(), c.GetName())
		}
		log.Infof("end purge federated %s joint resource for cluster %s", m.KeywordPlural(), c.GetName())
	}
	return nil
}

func (c *SCluster) DeleteAllComponents(ctx context.Context, userCred mcclient.TokenCredential) error {
	cs, err := c.GetClusterComponents()
	if err != nil {
		return err
	}
	for _, cp := range cs {
		comp, err := cp.GetComponent()
		if err != nil {
			return err
		}
		if err := cp.Detach(ctx, userCred); err != nil {
			return err
		}
		if err := comp.Delete(ctx, userCred); err != nil {
			return err
		}
	}
	return nil
}

func (c *SCluster) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return c.StartClusterDeleteTask(ctx, userCred, data.(*jsonutils.JSONDict), "")
}

func (c *SCluster) StartClusterDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	if err := c.SetStatus(userCred, api.ClusterStatusDeleting, ""); err != nil {
		return err
	}
	if err := client.GetClustersManager().RemoveClient(c.GetId()); err != nil {
		return errors.Wrap(err, "remove client before start delete task")
	}
	task, err := taskman.TaskManager.NewTask(ctx, "ClusterDeleteTask", c, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (c *SCluster) allowPerformAction(userCred mcclient.TokenCredential, action string) bool {
	return db.IsDomainAllowPerform(userCred, c, action)
}

func (c *SCluster) allowGetSpec(userCred mcclient.TokenCredential, spec string) bool {
	return db.IsDomainAllowGetSpec(userCred, c, spec)
}

func (c *SCluster) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return userCred.HasSystemAdminPrivilege()
}

func (c *SCluster) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query, input api.ClusterPurgeInput) (jsonutils.JSONObject, error) {
	if !input.Force {
		if err := c.ValidateDeleteCondition(ctx); err != nil {
			return nil, err
		}
	}
	return nil, c.StartClusterDeleteTask(ctx, userCred, input.JSON(input), "")
}

func (c *SCluster) AllowGetDetailsKubeconfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return c.allowGetSpec(userCred, "kubeconfig")
}

func (c *SCluster) GetRunningControlplaneMachine() (manager.IMachine, error) {
	return c.getControlplaneMachine(true)
}

func (c *SCluster) getControlplaneMachine(checkStatus bool) (manager.IMachine, error) {
	machines, err := c.GetMachines()
	if err != nil {
		return nil, err
	}
	if machines == nil {
		return nil, nil
	}
	for _, m := range machines {
		if m.IsControlplane() && m.IsFirstNode() {
			if !checkStatus {
				return m, nil
			}
			if m.IsRunning() {
				return m, nil
			} else {
				return nil, fmt.Errorf("Not found a running controlplane machine, status is %s", m.GetStatus())
			}
		}
	}
	return nil, fmt.Errorf("Not found a controlplane machine")
}

func (c *SCluster) GetControlplaneMachines() ([]manager.IMachine, error) {
	ms, err := c.GetMachines()
	if err != nil {
		return nil, err
	}
	ret := make([]manager.IMachine, 0)
	for _, m := range ms {
		if m.IsControlplane() {
			ret = append(ret, m)
		}
	}
	return ret, nil
}

func (c *SCluster) GetMachines() ([]manager.IMachine, error) {
	return manager.MachineManager().GetMachines(c.Id)
}

func (c *SCluster) GetMachinesByRole(role string) ([]manager.IMachine, error) {
	ms, err := c.GetMachines()
	if err != nil {
		return nil, err
	}
	ret := make([]manager.IMachine, 0)
	for _, m := range ms {
		if m.GetRole() == role {
			ret = append(ret, m)
		}
	}
	return ret, nil
}

func (c *SCluster) getKeyPairByUser(user string) (*SX509KeyPair, error) {
	return ClusterX509KeyPairManager.GetKeyPairByClusterUser(c.GetId(), user)
}

func (c *SCluster) GetCAKeyPair() (*SX509KeyPair, error) {
	return c.getKeyPairByUser(api.ClusterCA)
}

func (c *SCluster) GetEtcdCAKeyPair() (*SX509KeyPair, error) {
	return c.getKeyPairByUser(api.EtcdCA)
}

func (c *SCluster) GetFrontProxyCAKeyPair() (*SX509KeyPair, error) {
	return c.getKeyPairByUser(api.FrontProxyCA)
}

func (c *SCluster) GetSAKeyPair() (*SX509KeyPair, error) {
	return c.getKeyPairByUser(api.ServiceAccountCA)
}

func (c *SCluster) GetKubeconfig() (string, error) {
	if len(c.Kubeconfig) != 0 {
		return c.Kubeconfig, nil
	}
	//kubeconfig, err := c.GetDriver().GetKubeconfig(c)
	kubeconfig, err := c.GetKubeconfigByCerts()
	if err != nil {
		return "", err
	}
	return kubeconfig, c.SetKubeconfig(kubeconfig)
}

func (c *SCluster) GetClientV2() (*clientv2.Client, error) {
	kubeconfig, err := c.GetKubeconfig()
	if err != nil {
		return nil, err
	}
	return clientv2.NewClient(kubeconfig)
}

func (c *SCluster) GetKubeconfigByCerts() (string, error) {
	caKpObj, err := c.GetCAKeyPair()
	if err != nil {
		return "", errors.Wrap(err, "Get CA key pair")
	}
	caKp := caKpObj.ToKeyPair()
	cert, err := certificates.DecodeCertPEM(caKp.Cert)
	if err != nil {
		return "", errors.Wrap(err, "failed to decode CA Cert")
	} else if cert == nil {
		return "", errors.Errorf("certificate not found")
	}

	key, err := certificates.DecodePrivateKeyPEM(caKp.Key)
	if err != nil {
		return "", errors.Wrap(err, "failed to decode private key")
	} else if key == nil {
		return "", errors.Errorf("key not foudn in status")
	}
	controlPlaneURL, err := c.GetControlPlaneUrl()
	if err != nil {
		return "", errors.Wrap(err, "failed to get controlPlaneURL")
	}

	cfg, err := certificates.NewKubeconfig(c.GetName(), controlPlaneURL, cert, key)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate a kubeconfig")
	}

	yaml, err := clientcmd.Write(*cfg)
	if err != nil {
		return "", errors.Wrap(err, "failed to serialize config to yaml")
	}

	return string(yaml), nil
}

func (c *SCluster) SetK8sVersion(ctx context.Context, version string) error {
	_, err := db.Update(c, func() error {
		c.Version = version
		return nil
	})
	return err
}

func (c *SCluster) SetKubeconfig(kubeconfig string) error {
	_, err := db.Update(c, func() error {
		c.Kubeconfig = kubeconfig
		return nil
	})
	return err
}

func (c *SCluster) GetDetailsKubeconfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	conf, err := c.GetKubeconfig()
	if err != nil {
		return nil, err
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewString(conf), "kubeconfig")
	return ret, nil
}

func (c *SCluster) GetAdminKubeconfig() (string, error) {
	return c.GetKubeconfig()
}

func setK8sConfigField(c *rest.Config, tr func(rt http.RoundTripper) http.RoundTripper) *rest.Config {
	if tr != nil {
		c.WrapTransport = tr
	}
	c.Timeout = time.Second * 30
	return c
}

func (c *SCluster) GetK8sClientConfig(kubeConfig []byte) (*rest.Config, error) {
	var config *rest.Config
	var err error
	if kubeConfig != nil {
		apiconfig, err := clientcmd.Load(kubeConfig)
		if err != nil {
			return nil, err
		}

		clientConfig := clientcmd.NewDefaultClientConfig(*apiconfig, &clientcmd.ConfigOverrides{})
		config, err = clientConfig.ClientConfig()
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.Errorf("kubeconfig value is nil")
	}
	if err != nil {
		return nil, errors.Errorf("create kubernetes config failed: %v", err)
	}
	return config, nil
}

func (c *SCluster) GetK8sRestConfig() (*rest.Config, error) {
	kubeconfig, err := c.GetAdminKubeconfig()
	if err != nil {
		return nil, err
	}
	config, err := c.GetK8sClientConfig([]byte(kubeconfig))
	if err != nil {
		return nil, err
	}
	return setK8sConfigField(config, func(rt http.RoundTripper) http.RoundTripper {
		switch rt.(type) {
		case *http.Transport:
			rt.(*http.Transport).DisableKeepAlives = true
		}
		return rt
	}), nil
}

func (c *SCluster) GetK8sClient() (*kubernetes.Clientset, error) {
	config, err := c.GetK8sRestConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func (c *SCluster) AllowPerformApplyAddons(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return c.allowPerformAction(userCred, "apply-addons")
}

func (c *SCluster) PerformApplyAddons(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if err := c.StartApplyAddonsTask(ctx, userCred, data.(*jsonutils.JSONDict), ""); err != nil {
		return nil, err
	}
	return nil, nil
}

func (c *SCluster) AllowGetDetailsAddons(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return c.AllowGetDetails(ctx, userCred, query)
}

func (c *SCluster) genAddonsManifestConfigByQuery(query api.ClusterGetAddonsInput) *api.ClusterAddonsManifestConfig {
	return &api.ClusterAddonsManifestConfig{
		Network: api.ClusterAddonNetworkConfig{
			EnableNativeIPAlloc: query.EnableNativeIPAlloc,
		},
	}
}

func (c *SCluster) GetDetailsAddons(ctx context.Context, userCred mcclient.TokenCredential, query api.ClusterGetAddonsInput) (jsonutils.JSONObject, error) {
	addons, err := c.GetDriver().GetAddonsManifest(c, c.genAddonsManifestConfigByQuery(query))
	if err != nil {
		return nil, err
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewString(addons), "addons")
	return ret, nil
}

func (c *SCluster) StartApplyAddonsTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ClusterApplyAddonsTask", c, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (c *SCluster) AllowPerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return c.allowPerformAction(userCred, "syncstatus")
}

func (c *SCluster) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, c.StartSyncStatus(ctx, userCred, "")
}

func (c *SCluster) StartSyncStatus(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	return c.GetDriver().StartSyncStatus(c, ctx, userCred, parentTaskId)
}

func (c *SCluster) AllowPerformAddMachines(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return c.allowPerformAction(userCred, "add-machines")
}

func (c *SCluster) ValidateAddMachines(ctx context.Context, userCred mcclient.TokenCredential, ms []api.CreateMachineData) ([]*api.CreateMachineData, error) {
	machines := make([]*api.CreateMachineData, len(ms))
	for i := range ms {
		machines[i] = &ms[i]
	}
	driver := c.GetDriver()
	imageRepo, err := c.GetImageRepository()
	if err != nil {
		return nil, err
	}
	info := &api.ClusterMachineCommonInfo{
		CloudregionId: c.CloudregionId,
		VpcId:         c.VpcId,
	}
	if err := driver.ValidateCreateMachines(ctx, userCred, c, info, imageRepo, machines); err != nil {
		return nil, err
	}
	return machines, nil
}

func (c *SCluster) PerformAddMachines(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ms := []api.CreateMachineData{}
	if err := data.Unmarshal(&ms, "machines"); err != nil {
		return nil, err
	}
	if !utils.IsInStringArray(c.Status, []string{api.ClusterStatusRunning, api.ClusterStatusInit}) {
		return nil, httperrors.NewNotAcceptableError("Cluster status is %s", c.Status)
	}

	machines, err := c.ValidateAddMachines(ctx, userCred, ms)
	if err != nil {
		return nil, err
	}

	return nil, c.StartCreateMachinesTask(ctx, userCred, machines, "")
}

func (c *SCluster) NeedControlplane() (bool, error) {
	ms, err := c.GetMachines()
	if err != nil {
		return false, errors.Wrapf(err, "get cluster %s machines", c.GetName())
	}
	if len(ms) == 0 {
		return true, nil
	}
	return false, nil
}

func (c *SCluster) StartCreateMachinesTask(ctx context.Context, userCred mcclient.TokenCredential, machines []*api.CreateMachineData, parentTaskId string) error {
	data := jsonutils.NewDict()
	data.Add(jsonutils.Marshal(machines), "machines")
	task, err := taskman.TaskManager.NewTask(ctx, "ClusterCreateMachinesTask", c, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (c *SCluster) CreateMachines(ctx context.Context, userCred mcclient.TokenCredential, ms []*api.CreateMachineData, task taskman.ITask) error {
	drv := c.GetDriver()
	machines, err := drv.CreateMachines(ctx, userCred, c, ms)
	if err != nil {
		return err
	}
	return drv.RequestDeployMachines(ctx, userCred, c, machines, task)
}

const (
	MachinesDeployIdsKey = "MachineIds"
)

func (c *SCluster) StartDeployMachinesTask(ctx context.Context, userCred mcclient.TokenCredential, machineIds []string, parentTaskId string) error {
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewStringArray(machineIds), MachinesDeployIdsKey)
	task, err := taskman.TaskManager.NewTask(ctx, "ClusterDeployMachinesTask", c, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (c *SCluster) AllowPerformDeleteMachines(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return c.allowPerformAction(userCred, "delete-machines")
}

func (c *SCluster) PerformDeleteMachines(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	machinesData, err := data.(*jsonutils.JSONDict).GetArray("machines")
	if err != nil {
		return nil, httperrors.NewInputParameterError("NotFound machines data: %v", err)
	}
	machines := []manager.IMachine{}
	for _, obj := range machinesData {
		id, err := obj.GetString()
		if err != nil {
			return nil, err
		}
		machineObj, err := manager.MachineManager().FetchMachineByIdOrName(userCred, id)
		if err != nil {
			return nil, httperrors.NewInputParameterError("Not found node by id: %s", id)
		}
		machines = append(machines, machineObj)
	}
	if len(machines) == 0 {
		return nil, httperrors.NewInputParameterError("Machines id is empty")
	}
	nowCnt, err := c.GetMachinesCount()
	if err != nil {
		return nil, err
	}
	// delete all machines
	if nowCnt == len(machines) {
		if err := c.CheckPVCEmpty(); err != nil {
			return nil, err
		}
	}
	driver := c.GetDriver()
	if err := driver.ValidateDeleteMachines(ctx, userCred, c, machines); err != nil {
		return nil, err
	}
	return nil, c.StartDeleteMachinesTask(ctx, userCred, machines, data.(*jsonutils.JSONDict), "")
}

func (c *SCluster) StartDeleteMachinesTask(ctx context.Context, userCred mcclient.TokenCredential, ms []manager.IMachine, data *jsonutils.JSONDict, parentTaskId string) error {
	if data == nil {
		data = jsonutils.NewDict()
	}
	mids := []jsonutils.JSONObject{}
	for _, m := range ms {
		m.SetStatus(userCred, api.MachineStatusDeleting, "ClusterDeleteMachinesTask")
		mids = append(mids, jsonutils.NewString(m.GetId()))
	}
	data.Set("machines", jsonutils.NewArray(mids...))
	task, err := taskman.TaskManager.NewTask(ctx, "ClusterDeleteMachinesTask", c, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (c *SCluster) GetControlPlaneUrl() (string, error) {
	apiServerEndpoint, err := c.GetAPIServerPublicEndpoint()
	if err != nil {
		return "", errors.Wrapf(err, "GetAPIServerEndpoint")
	}
	return fmt.Sprintf("https://%s:6443", apiServerEndpoint), nil
}

func (c *SCluster) GetAPIServer() (string, error) {
	if len(c.ApiServer) != 0 {
		return c.ApiServer, nil
	}

	apiServer, err := c.GetControlPlaneUrl()
	if err != nil {
		return "", err
	}
	return apiServer, c.SetAPIServer(apiServer)
}

func (c *SCluster) SetAPIServer(apiServer string) error {
	_, err := db.Update(c, func() error {
		c.ApiServer = apiServer
		return nil
	})
	return err
}

func (c *SCluster) GetAPIServerPublicEndpoint() (string, error) {
	if c.IsInClassicNetwork() {
		return c.GetAPIServerInternalEndpoint()
	}
	m, err := c.getControlplaneMachine(false)
	if err != nil {
		return "", errors.Wrap(err, "get controlplane machine")
	}
	ip, err := m.GetEIP()
	if err != nil {
		return "", errors.Wrapf(err, "get controlplane machine %s EIP", m.GetName())
	}
	return ip, nil
}

// TODO: support use loadbalancer
func (c *SCluster) GetAPIServerInternalEndpoint() (string, error) {
	m, err := c.getControlplaneMachine(false)
	if err != nil {
		return "", errors.Wrap(err, "get controlplane machine")
	}
	ip, err := m.GetPrivateIP()
	if err != nil {
		return "", errors.Wrapf(err, "get controlplane machine %s private ip", m.GetName())
	}
	return ip, nil
}

func (c *SCluster) GetPodCidr() string {
	return c.PodCidr
}

func (c *SCluster) GetServiceCidr() string {
	return c.ServiceCidr
}

func (c *SCluster) GetServiceDomain() string {
	return c.ServiceDomain
}

func (c *SCluster) GetVersion() string {
	return c.Version
}

func (c *SCluster) GetClusterComponents() ([]SClusterComponent, error) {
	cs := make([]SClusterComponent, 0)
	q := ClusterComponentManager.Query().Equals("cluster_id", c.GetId())
	if err := db.FetchModelObjects(ClusterComponentManager, q, &cs); err != nil {
		return nil, err
	}
	return cs, nil
}

func (c *SCluster) GetComponents() ([]*SComponent, error) {
	cs, err := c.GetClusterComponents()
	if err != nil {
		return nil, err
	}
	ret := make([]*SComponent, 0)
	for _, cc := range cs {
		obj, err := cc.GetComponent()
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				continue
			}
			return nil, err
		}
		ret = append(ret, obj)
	}
	return ret, nil
}

func (c *SCluster) GetComponentByTypeNoError(cType string) (*SComponent, error) {
	cs, err := c.GetComponents()
	if err != nil {
		return nil, err
	}
	for _, comp := range cs {
		if comp.Type == cType {
			return comp, nil
		}
	}
	return nil, nil
}

func (c *SCluster) GetComponentByType(cType string) (*SComponent, error) {
	comp, err := c.GetComponentByTypeNoError(cType)
	if err != nil {
		return nil, err
	}
	if comp == nil {
		return nil, httperrors.NewNotFoundError("not found component by type %q", cType)
	}
	return comp, nil
}

func (c *SCluster) EnableComponent(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input *api.ComponentCreateInput) error {
	comp, err := c.GetComponentByTypeNoError(input.Type)
	if err != nil {
		return err
	}
	if comp != nil {
		return comp.DoEnable(ctx, userCred, nil, "")
	}

	defer lockman.ReleaseObject(ctx, c)
	lockman.LockObject(ctx, c)

	comp, err = ComponentManager.CreateByCluster(ctx, userCred, c, input)
	if err != nil {
		return err
	}
	return nil
}

func (c *SCluster) AllowGetDetailsComponentsStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsProjectAllowGetSpec(userCred, c, "components-status")
}

func (c *SCluster) GetDetailsComponentsStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*api.ComponentsStatus, error) {
	return c.GetComponentsStatus()
}

func (c *SCluster) GetComponentsStatus() (*api.ComponentsStatus, error) {
	status := new(api.ComponentsStatus)
	drvs := ComponentManager.GetDrivers()
	for _, drv := range drvs {
		comp, err := c.GetComponentByTypeNoError(drv.GetType())
		if err != nil {
			return nil, errors.Wrapf(err, "cluster get component by type: %s", drv.GetType())
		}
		if comp == nil {
			// not created
			if err := drv.FetchStatus(c, comp, status); err != nil {
				return nil, err
			}
		} else {
			if err := drv.FetchStatus(c, comp, status); err != nil {
				return nil, err
			}
		}
	}
	return status, nil
}

func (c *SCluster) AllowGetDetailsComponentSetting(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsProjectAllowGetSpec(userCred, c, "component-setting")
}

func (c *SCluster) GetDetailsComponentSetting(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !query.Contains("type") {
		return nil, httperrors.NewInputParameterError("type not provided")
	}
	cType, _ := query.GetString("type")
	cs, err := c.GetComponentByType(cType)
	if err != nil {
		return nil, err
	}
	asHelmValues, _ := query.Bool("as_helm_values")
	if !asHelmValues {
		return cs.Settings, nil
	}

	settings, err := cs.GetSettings()
	if err != nil {
		return nil, err
	}

	driver, err := cs.GetDriver()
	if err != nil {
		return nil, err
	}

	vals, err := driver.GetHelmValues(c, settings)
	if err != nil {
		return nil, errors.Wrap(err, "get helm values")
	}

	return jsonutils.Marshal(vals), nil
}

func (c *SCluster) AllowPerformEnableComponent(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return c.allowPerformAction(userCred, "enable-component")
}

func (c *SCluster) PerformEnableComponent(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.ComponentCreateInput) (jsonutils.JSONObject, error) {
	if err := c.EnableComponent(ctx, userCred, input); err != nil {
		log.Errorf("enable comp error: %v", err)
		return nil, err
	}
	comp, err := c.GetComponentByType(input.Type)
	if err != nil {
		return nil, err
	}
	ret, err := comp.GetExtraDetails(ctx, userCred, query, false)
	if err != nil {
		return nil, err
	}
	return jsonutils.Marshal(ret), nil
}

func (c *SCluster) AllowPerformDisableComponent(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return c.allowPerformAction(userCred, "disable-component")
}

func (c *SCluster) PerformDisableComponent(ctx context.Context, userCred mcclient.TokenCredential, query, input api.ComponentDeleteInput) (jsonutils.JSONObject, error) {
	comp, err := c.GetComponentByType(input.Type)
	if err != nil {
		return nil, err
	}
	return nil, comp.DoDisable(ctx, userCred, input.JSON(input), "")
}

func (c *SCluster) AllowPerformDeleteComponent(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return c.allowPerformAction(userCred, "delete-component")
}

func (c *SCluster) PerformDeleteComponent(ctx context.Context, userCred mcclient.TokenCredential, query, input *api.ComponentDeleteInput) (jsonutils.JSONObject, error) {
	comp, err := c.GetComponentByType(input.Type)
	if err != nil {
		return nil, err
	}
	return nil, comp.DoDelete(ctx, userCred, input.JSON(input), "")
}

func (c *SCluster) AllowPerformUpdateComponent(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return c.allowPerformAction(userCred, "update-component")
}

func (c *SCluster) PerformUpdateComponent(ctx context.Context, userCred mcclient.TokenCredential, query, input *api.ComponentUpdateInput) (jsonutils.JSONObject, error) {
	comp, err := c.GetComponentByType(input.Type)
	if err != nil {
		return nil, err
	}
	drv, err := comp.GetDriver()
	if err != nil {
		return nil, err
	}
	if err := drv.ValidateUpdateData(ctx, userCred, c, input); err != nil {
		return nil, err
	}
	if err := comp.DoUpdate(ctx, userCred, input); err != nil {
		return nil, err
	}
	return nil, nil
}

/*func (c *SCluster) enableSystemMonitorStack(ctx context.Context, userCred mcclient.TokenCredential, s3Config *api.ObjectStoreConfig) error {
	stackSetting := &api.ComponentSettingMonitor{
		Grafana: &api.ComponentSettingMonitorGrafana{
			Storage: &api.ComponentStorage{
				Enabled: true,
				SizeMB:  4096,
			},
		},
		Loki: &api.ComponentSettingMonitorLoki{
			ObjectStoreConfig: s3Config,
		},
		Prometheus: &api.ComponentSettingMonitorPrometheus{
			Storage: &api.ComponentStorage{
				Enabled: true,
				SizeMB:  10240,
			},
			ThanosSidecar: &api.ComponentSettingMonitorPrometheusThanos{},
		},
	}
	input := &api.ComponentCreateInput{
		Name:    MonitorReleaseName,
		Type:    api.ClusterComponentMonitor,
		Cluster: c.GetId(),
		ComponentSettings: api.ComponentSettings{
			Namespace: MonitorNamespace,
			Monitor:   stackSetting,
		},
	}
	if err := c.EnableComponent(ctx, userCred, input); err != nil {
		return errors.Wrap(err, "enable component monitor stack")
	}

	return nil
}*/

func (c *SCluster) prepareStartSync() error {
	if c.GetStatus() != api.ClusterStatusRunning {
		return errors.Errorf("Cluster status is %s", c.GetStatus())
	}
	return nil
}

func (m *SClusterManager) WaitFullSynced() error {
	ctx := context.TODO()
	userCred := GetAdminCred()
	return m.startAutoSyncTask(ctx, userCred, true)
}

func (m *SClusterManager) startAutoSyncTask(ctx context.Context, userCred mcclient.TokenCredential, wait bool) error {
	clusters, err := m.GetRunningClusters()
	if err != nil {
		return errors.Wrap(err, "Start auto sync cluster task get running clusters: %v")
	}
	errs := make([]error, 0)
	for _, cls := range clusters {
		if wait {
			waitCh := make(chan error, 0)
			cls.(*SCluster).SubmitSyncTask(ctx, userCred, waitCh)
			if err := <-waitCh; err != nil {
				errs = append(errs, err)
			}
		} else {
			cls.(*SCluster).SubmitSyncTask(ctx, userCred, nil)
		}
	}
	return errors.NewAggregate(errs)
}

func (m *SClusterManager) StartAutoSyncTask(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	m.startAutoSyncTask(ctx, userCred, false)
}

func (c *SCluster) AllowPerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, c, "sync")
}

func (c *SCluster) PerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	input := new(api.ClusterSyncInput)
	data.Unmarshal(input)
	if c.CanSync() || input.Force {
		c.StartSyncTask(ctx, userCred, nil, "")
	}
	return nil, nil
}

func (c *SCluster) StartSyncTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ClusterSyncTask", c, userCred, data, parentId, "")
	if err != nil {
		return errors.Wrap(err, "New ClusterSyncTask")
	}
	task.ScheduleRun(nil)
	return nil
}

func (c *SCluster) SyncCallSyncTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	waitCh := make(chan error)
	c.SubmitSyncTask(ctx, userCred, waitCh)
	return <-waitCh
}

func (c *SCluster) SubmitSyncTask(ctx context.Context, userCred mcclient.TokenCredential, waitChan chan error) {
	if err := c.DisableBidirectionalSync(); err != nil {
		if waitChan != nil {
			log.Errorf("DisableBidirectionalSync before submic sync error: %s", err)
			waitChan <- err
		}
		return
	}
	RunSyncClusterTask(func() {
		log.Infof("start sync cluster %s", c.GetName())
		if err := c.prepareStartSync(); err != nil {
			log.Errorf("sync cluster task error: %v", err)
			if waitChan != nil {
				waitChan <- err
			}
			return
		}
		if err := c.MarkSyncing(c, userCred); err != nil {
			log.Errorf("Mark cluster %s syncing error: %v", c.GetId(), err)
			if waitChan != nil {
				waitChan <- err
			}
			return
		}

		for _, man := range GetClusterManager().GetSubManagers() {
			err := SyncClusterResources(ctx, userCred, c, man)
			if err != nil {
				log.Errorf("Sync %s error: %v", man.KeywordPlural(), err)
				c.MarkErrorSync(ctx, c, err)
				if waitChan != nil {
					waitChan <- err
				}
				return
			}
		}
		if err := c.MarkEndSync(ctx, userCred, c); err != nil {
			log.Errorf("mark cluster %s sync end error: %v", c.GetId(), err)
			if waitChan != nil {
				waitChan <- err
			}
			return
		}
		if err := c.EnableBidirectionalSync(); err != nil {
			log.Errorf("EnableBidirectionalSync cluster %s sync end error: %v", c.GetId(), err)
			if waitChan != nil {
				waitChan <- err
			}
			return
		}
		if waitChan != nil {
			waitChan <- nil
			return
		}
	})
}

func (c *SCluster) GetK8sResourceManager(kindName string) manager.IK8sResourceManager {
	return GetK8sResourceManagerByKind(kindName)
}

type sClusterUsage struct {
	Id string
}

func (m *SClusterManager) usageClusters(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, isSystem bool) ([]sClusterUsage, error) {
	q := m.Query("id", "is_system")
	if isSystem {
		q = q.IsTrue("is_system")
	} else {
		q = q.Filter(sqlchemy.OR(
			sqlchemy.IsNullOrEmpty(q.Field("is_system")),
			sqlchemy.IsFalse(q.Field("is_system"))))
	}
	switch scope {
	case rbacutils.ScopeSystem:
		// do nothing
	case rbacutils.ScopeDomain:
		q = q.Equals("domain_id", ownerId.GetProjectDomainId())
	}
	var clusters []sClusterUsage
	if err := q.All(&clusters); err != nil {
		return nil, errors.Wrap(err, "query cluster usage")
	}
	return clusters, nil
}

func (m *SClusterManager) Usage(scope rbacutils.TRbacScope, ownerId mcclient.IIdentityProvider, isSystem bool) (*api.ClusterUsage, error) {
	usage := new(api.ClusterUsage)
	clusters, err := m.usageClusters(scope, ownerId, isSystem)
	if err != nil {
		return nil, err
	}
	usage.Count = int64(len(clusters))
	nodeUsage, err := GetNodeManager().Usage(clusters)
	if err != nil {
		return nil, errors.Wrap(err, "get node usage")
	}
	usage.Node = nodeUsage
	return usage, nil
}

// GetRemoteClient get remote kubernetes wrapped client
func (c *SCluster) GetRemoteClient() (*client.ClusterManager, error) {
	return client.GetManagerByCluster(c)
}

func (c *SCluster) GetDetailsApiResources(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (api.ClusterAPIGroupResources, error) {
	rCli, err := c.GetRemoteClient()
	if err != nil {
		return nil, errors.Wrap(err, "get remote kubernetes client")
	}
	dCli := rCli.KubeClient.GetClientset().Discovery()
	// dCli, err := rCli.ClientV2.K8S().ToDiscoveryClient()
	// if err != nil {
	// return nil, errors.Wrap(err, "get discoveryClient")
	// }
	lists, err := dCli.ServerPreferredResources()
	if err != nil {
		// return nil, errors.Wrap(err, "get server preferred resources")
		log.Errorf("get server preferred resources error: %v", err)
	}
	resources := []api.ClusterAPIGroupResource{}
	// ref code from kubernetes/pkg/kubectl/cmd/apiresources/apiresources.go
	for _, list := range lists {
		if len(list.APIResources) == 0 {
			continue
		}
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			continue
		}
		for _, resource := range list.APIResources {
			if len(resource.Verbs) == 0 {
				continue
			}
			resources = append(resources, api.ClusterAPIGroupResource{
				APIGroup:    gv.Group,
				APIResource: resource,
			})
		}
	}
	return resources, nil
}

func (c *SCluster) GetDetailsClusterUsers(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (api.ClusterUsers, error) {
	if c.Distribution != api.ImportClusterDistributionOpenshift {
		return nil, nil
	}
	config, err := c.GetK8sRestConfig()
	if err != nil {
		return nil, errors.Wrap(err, "get k8s restconfig")
	}
	drv := c.GetDriver()
	return drv.GetClusterUsers(c, config)
}

func (c *SCluster) GetDetailsClusterUserGroups(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (api.ClusterUserGroups, error) {
	if c.Distribution != api.ImportClusterDistributionOpenshift {
		return nil, nil
	}
	config, err := c.GetK8sRestConfig()
	if err != nil {
		return nil, errors.Wrap(err, "get k8s restconfig")
	}
	drv := c.GetDriver()
	return drv.GetClusterUserGroups(c, config)
}

func (c *SCluster) EnableBidirectionalSync() error {
	cli, err := c.GetRemoteClient()
	if err != nil {
		return errors.Wrap(err, "GetRemoteClient")
	}
	cli.GetHandler().EnableBidirectionalSync()
	return nil
}

func (c *SCluster) DisableBidirectionalSync() error {
	cli, err := c.GetRemoteClient()
	if err != nil {
		return errors.Wrap(err, "GetRemoteClient")
	}
	cli.GetHandler().DisableBidirectionalSync()
	return nil
}

func (c *SCluster) IsInClassicNetwork() bool {
	return c.VpcId == computeapi.DEFAULT_VPC_ID
}
