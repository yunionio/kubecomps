package models

import (
	"context"

	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
)

const (
	MonitorMinioReleaseName = "monitor-minio"
)

var (
	MonitorMinioComponentManager *SMonitorMinioComponentManager
)

func init() {
	MonitorMinioComponentManager = NewMonitorMinioComponentManager()
	ComponentManager.RegisterDriver(newComponentDriverMonitorMinio())
}

func NewMonitorMinioComponentManager() *SMonitorMinioComponentManager {
	bMan := NewMinioBaseComponentManager(
		"kubecomponentmonitorminio", "kubecomponentmonitorminios",
		MonitorMinioReleaseName, MonitorNamespace,
		SMinioComponent{},
	)
	man := new(SMonitorMinioComponentManager)
	man.SMinioBaseComponentManager = *bMan
	man.SetVirtualObject(man)
	return man
}

type componentDriverMonitorMinio struct {
	*componentDriverMinioBase
}

func newComponentDriverMonitorMinio() IComponentDriver {
	return componentDriverMonitorMinio{
		componentDriverMinioBase: newComponentDriverMinioBase(api.ClusterComponentMonitorMinio, MonitorMinioComponentManager),
	}
}

func (c componentDriverMonitorMinio) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, input *api.ComponentCreateInput) error {
	return c.validateSetting(ctx, userCred, cluster, input.MonitorMinio)
}

func (c componentDriverMonitorMinio) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, input *api.ComponentUpdateInput) error {
	return c.validateSetting(ctx, userCred, cluster, input.MonitorMinio)
}

func (c componentDriverMonitorMinio) GetCreateSettings(input *api.ComponentCreateInput) (*api.ComponentSettings, error) {
	return c.getCreateSettings(input, MonitorNamespace)
}

func (c componentDriverMonitorMinio) GetUpdateSettings(oldSetting *api.ComponentSettings, input *api.ComponentUpdateInput) (*api.ComponentSettings, error) {
	oldSetting.MonitorMinio = input.MonitorMinio
	return oldSetting, nil
}

func (c componentDriverMonitorMinio) DoEnable(cluster *SCluster, setting *api.ComponentSettings) error {
	return DoEnableMinio(MonitorMinioComponentManager, cluster, setting)
}

func (c componentDriverMonitorMinio) DoDisable(cluster *SCluster, setting *api.ComponentSettings) error {
	return DoDisableMinio(MonitorMinioComponentManager, cluster, setting)
}

func (c componentDriverMonitorMinio) DoUpdate(cluster *SCluster, setting *api.ComponentSettings) error {
	return DoUpdateMinio(MonitorMinioComponentManager, cluster, setting)
}

func (c componentDriverMonitorMinio) FetchStatus(cluster *SCluster, comp *SComponent, status *api.ComponentsStatus) error {
	if status.MonitorMinio == nil {
		status.MonitorMinio = new(api.ComponentStatus)
	}
	return c.fetchStatus(cluster, comp, status.MonitorMinio)
}

type SMonitorMinioComponentManager struct {
	SMinioBaseComponentManager
}

func (m SMonitorMinioComponentManager) CreateHelmResource(cluster *SCluster, setting *api.ComponentSettings) error {
	return CreateMinioHelmResource(m.GetHelmManager(), cluster, setting, setting.MonitorMinio)
}

func (m SMonitorMinioComponentManager) UpdateHelmResource(cluster *SCluster, setting *api.ComponentSettings) error {
	return UpdateMinioHelmResource(m.GetHelmManager(), cluster, setting, setting.MonitorMinio)
}

func (m SMonitorMinioComponentManager) DeleteHelmResource(cluster *SCluster, setting *api.ComponentSettings) error {
	return DeleteMinioHelmResource(m.GetHelmManager(), cluster)
}

func (m SMonitorMinioComponentManager) GetHelmValues(cluster *SCluster, setting *api.ComponentSettings) (map[string]interface{}, error) {
	return GetMinioHelmValues(cluster, setting, setting.MonitorMinio)
}
