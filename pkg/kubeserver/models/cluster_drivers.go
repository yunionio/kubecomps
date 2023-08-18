package models

import (
	"context"
	"fmt"

	"k8s.io/client-go/rest"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

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
	PreCheck(s *mcclient.ClientSession, data jsonutils.JSONObject) (*api.ClusterPreCheckResp, error)

	IClusterDriverMethods
}

type ClusterHelmChartInstallOption struct {
	EmbedChartName string
	ReleaseName    string
	Namespace      string
	Values         map[string]interface{}
}

func (o ClusterHelmChartInstallOption) Validate() error {
	for k, v := range map[string]string{
		"embed_chart_name": o.EmbedChartName,
		"release_name":     o.ReleaseName,
		"namespace":        o.Namespace,
	} {
		if v == "" {
			return errors.Errorf("%s is empty", k)
		}
	}
	return nil
}

type IClusterDriverMethods interface {
	// GetUsableInstances return usable instances for cluster
	GetUsableInstances(s *mcclient.ClientSession) ([]api.UsableInstance, error)

	NeedCreateMachines() bool

	ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.ClusterCreateInput) error
	// RequestDeployMachines run ansible deploy machines as kubernetes nodes
	RequestDeployMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, action api.ClusterDeployAction, machines []manager.IMachine, skipDownloads bool, task taskman.ITask) error
	GetKubesprayConfig(ctx context.Context, cluster *SCluster) (*api.ClusterKubesprayConfig, error)

	ValidateDeleteCondition() error
	ValidateDeleteMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, machines []manager.IMachine) error
	RequestDeleteMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, machines []manager.IMachine, task taskman.ITask) error

	ValidateCreateMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, info *api.ClusterMachineCommonInfo, imageRepo *api.ImageRepository, data []*api.CreateMachineData) error

	GetAddonsHelmCharts(cluster *SCluster, conf *api.ClusterAddonsManifestConfig) ([]*ClusterHelmChartInstallOption, error)

	// GetAddonsManifest return addons yaml manifest to be applied to cluster
	GetAddonsManifest(cluster *SCluster, conf *api.ClusterAddonsManifestConfig) (string, error)
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
		panic(fmt.Sprintf("cluster driver provider %s, resource type %s driver register error: %v", provider, resType, err))
	}
}

func GetDriverWithError(
	mode api.ModeType,
	provider api.ProviderType,
	resType api.ClusterResourceType,
) (IClusterDriver, error) {
	drv, err := clusterDrivers.Get(string(mode), string(provider), string(resType))
	if err != nil {
		if errors.Cause(err) == drivers.ErrDriverNotFound {
			return nil, httperrors.NewNotFoundError("Not found support driver by %s/%s/%s", mode, provider, resType)
		}
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
