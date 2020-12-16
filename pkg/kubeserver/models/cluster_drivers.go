package models

import (
	"context"

	"k8s.io/client-go/rest"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers"
	"yunion.io/x/kubecomps/pkg/kubeserver/models/manager"
)

type IClusterDriver interface {
	GetMode() api.ModeType
	GetProvider() api.ProviderType
	GetResourceType() api.ClusterResourceType

	// GetK8sVersions return current cluster k8s versions supported
	GetK8sVersions() []string
	// GetUsableInstances return usable instances for cluster
	GetUsableInstances(session *mcclient.ClientSession) ([]api.UsableInstance, error)
	// GetKubeconfig get current cluster kubeconfig
	GetKubeconfig(cluster *SCluster) (string, error)

	ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.ClusterCreateInput) error
	ValidateDeleteCondition() error
	ValidateDeleteMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, machines []manager.IMachine) error
	RequestDeleteMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, machines []manager.IMachine, task taskman.ITask) error

	// CreateClusterResource create cluster resource to global k8s cluster
	CreateClusterResource(man *SClusterManager, data *api.ClusterCreateInput) error
	ValidateCreateMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, info *api.ClusterMachineCommonInfo, imageRepo *api.ImageRepository, data []*api.CreateMachineData) error
	// CreateMachines create machines record in db
	CreateMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, data []*api.CreateMachineData) ([]manager.IMachine, error)
	// RequestDeployMachines deploy machines after machines created
	RequestDeployMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, machines []manager.IMachine, task taskman.ITask) error
	// GetAddonsManifest return addons yaml manifest to be applied to cluster
	GetAddonsManifest(cluster *SCluster, conf *api.ClusterAddonsManifestConfig) (string, error)
	// StartSyncStatus start cluster sync status task
	StartSyncStatus(cluster *SCluster, ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error

	// need generate kubeadm certificates
	NeedGenerateCertificate() bool
	// NeedCreateMachines make this driver create machines models
	NeedCreateMachines() bool

	GetMachineDriver(resourceType api.MachineResourceType) IMachineDriver
	// GetClusterUsers query users resource from remote k8s cluster
	GetClusterUsers(cluster *SCluster, restCfg *rest.Config) ([]api.ClusterUser, error)
	// GetClusterUserGroups query groups resource from remote k8s cluster
	GetClusterUserGroups(cluster *SCluster, restCfg *rest.Config) ([]api.ClusterUserGroup, error)
}

var clusterDrivers *drivers.DriverManager

func init() {
	clusterDrivers = drivers.NewDriverManager("")
}

func RegisterClusterDriver(driver IClusterDriver) {
	modeType := driver.GetMode()
	resType := driver.GetResourceType()
	provider := driver.GetProvider()
	err := clusterDrivers.Register(driver,
		string(modeType),
		string(provider),
		string(resType))
	if err != nil {
		log.Fatalf("cluster driver provider %s, resource type %s driver register error: %v", provider, resType, err)
	}
}

func GetDriverWithError(
	mode api.ModeType,
	provider api.ProviderType,
	resType api.ClusterResourceType,
) (IClusterDriver, error) {
	drv, err := clusterDrivers.Get(string(mode), string(provider), string(resType))
	if err != nil {
		return nil, err
	}
	return drv.(IClusterDriver), nil
}

func GetClusterDriver(mode api.ModeType, provider api.ProviderType, resType api.ClusterResourceType) IClusterDriver {
	drv, err := GetDriverWithError(mode, provider, resType)
	if err != nil {
		log.Fatalf("Get driver cluster provider: %s, resource type: %s error: %v", provider, resType, err)
	}
	return drv.(IClusterDriver)
}
