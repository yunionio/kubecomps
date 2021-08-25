package models

import (
	"context"

	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
)

const (
	MinioNamespace   = "onecloud-minio"
	MinioReleaseName = "minio"
)

var (
	MinioComponentManager *SMinioComponentManager
)

func init() {
	MinioComponentManager = NewMinioComponentManager()
	ComponentManager.RegisterDriver(newComponentDriverMinio())
}

type SMinioComponentManager struct {
	SMinioBaseComponentManager
}

type SMinioComponent struct {
	SMinioBaseComponent
}

func NewMinioComponentManager() *SMinioComponentManager {
	bMan := NewMinioBaseComponentManager(
		"kubecomponentminio", "kubecomponentminios",
		MinioReleaseName, MinioNamespace,
		SMinioComponent{},
	)
	man := &SMinioComponentManager{
		SMinioBaseComponentManager: *bMan,
	}
	man.SetVirtualObject(man)
	return man
}

type componentDriverMinio struct {
	*componentDriverMinioBase
}

func newComponentDriverMinio() IComponentDriver {
	return componentDriverMinio{
		componentDriverMinioBase: newComponentDriverMinioBase(api.ClusterComponentMinio, MinioComponentManager),
	}
}

func (c componentDriverMinio) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, input *api.ComponentCreateInput) error {
	return c.validateSetting(ctx, userCred, cluster, input.Minio)
}

func (c componentDriverMinio) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, input *api.ComponentUpdateInput) error {
	return c.validateSetting(ctx, userCred, cluster, input.Minio)
}

func (c componentDriverMinio) GetCreateSettings(input *api.ComponentCreateInput) (*api.ComponentSettings, error) {
	return c.getCreateSettings(input, MinioNamespace)
}

func (c componentDriverMinio) GetUpdateSettings(oldSetting *api.ComponentSettings, input *api.ComponentUpdateInput) (*api.ComponentSettings, error) {
	oldSetting.Minio = input.Minio
	return oldSetting, nil
}

func (c componentDriverMinio) DoEnable(cluster *SCluster, setting *api.ComponentSettings) error {
	return DoEnableMinio(MinioComponentManager, cluster, setting)
}

func (c componentDriverMinio) DoDisable(cluster *SCluster, setting *api.ComponentSettings) error {
	return DoDisableMinio(MinioComponentManager, cluster, setting)
}

func (c componentDriverMinio) DoUpdate(cluster *SCluster, setting *api.ComponentSettings) error {
	return DoUpdateMinio(MinioComponentManager, cluster, setting)
}

func (c componentDriverMinio) FetchStatus(cluster *SCluster, comp *SComponent, status *api.ComponentsStatus) error {
	if status.Minio == nil {
		status.Minio = new(api.ComponentStatus)
	}
	return c.fetchStatus(cluster, comp, status.Minio)
}

func (m SMinioComponentManager) CreateHelmResource(cluster *SCluster, setting *api.ComponentSettings) error {
	return CreateMinioHelmResource(m.GetHelmManager(), cluster, setting.Minio)
}

func (m SMinioComponentManager) DeleteHelmResource(cluster *SCluster, setting *api.ComponentSettings) error {
	return DeleteMinioHelmResource(m.GetHelmManager(), cluster)
}

func (m SMinioComponentManager) UpdateHelmResource(cluster *SCluster, setting *api.ComponentSettings) error {
	return UpdateMinioHelmResource(m.HelmComponentManager, cluster, setting.Minio)
}

func (m SMinioComponentManager) GetHelmValues(cluster *SCluster, setting *api.ComponentSettings) (map[string]interface{}, error) {
	return GetMinioHelmValues(cluster, setting.Minio)
}
