package models

import (
	"context"
	"fmt"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/embed"
	"yunion.io/x/kubecomps/pkg/kubeserver/templates/components"
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
	SComponentManager
	HelmComponentManager
}

type SMinioComponent struct {
	SComponent
}

func NewMinioComponentManager() *SMinioComponentManager {
	man := new(SMinioComponentManager)
	man.SComponentManager = *NewComponentManager(SMinioComponent{},
		"kubecomponentminio",
		"kubecomponentminios",
	)
	man.HelmComponentManager = *NewHelmComponentManager(MinioNamespace, MinioReleaseName, embed.MINIO_8_0_6_TGZ)
	man.SetVirtualObject(man)
	return man
}

type componentDriverMinio struct {
	baseComponentDriver
}

func newComponentDriverMinio() IComponentDriver {
	return new(componentDriverMinio)
}

func (c componentDriverMinio) GetType() string {
	return api.ClusterComponentMinio
}

func (c componentDriverMinio) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, input *api.ComponentCreateInput) error {
	return c.validateSetting(ctx, userCred, cluster, input.Minio)
}

func (c componentDriverMinio) validateSetting(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, conf *api.ComponentSettingMinio) error {
	if conf == nil {
		return httperrors.NewNotEmptyError("minio config is empty")
	}

	if conf.Mode != api.ComponentMinoModeDistributed && conf.Mode != api.ComponentMinoModeStandalone {
		return httperrors.NewInputParameterError("not support mode %s", conf.Mode)
	}
	if conf.Mode == api.ComponentMinoModeStandalone {
		if conf.Replicas != 1 {
			return httperrors.NewInputParameterError("standalone mode replica should be 1")
		}
	} else {
		if conf.Replicas < 4 {
			return httperrors.NewInputParameterError("distributed mode replicas must >= 4, input %d", conf.Replicas)
		}
		if conf.Zones == 0 {
			conf.Zones = 1
		}
		if conf.DrivesPerNode == 0 {
			conf.DrivesPerNode = 1
		}
	}

	if conf.AccessKey == "" {
		return httperrors.NewNotEmptyError("access key is empty")
	}
	if len(conf.AccessKey) < 3 {
		return httperrors.NewInputParameterError("access key length should be at least 3")
	}

	if conf.SecretKey == "" {
		return httperrors.NewNotEmptyError("secret key is empty")
	}
	if len(conf.SecretKey) < 8 {
		return httperrors.NewInputParameterError("secret key length should be at least 8")
	}

	if conf.MountPath == "" {
		conf.MountPath = "/export"
	}

	if err := c.validateStorage(userCred, cluster, &conf.Storage); err != nil {
		return err
	}

	return nil
}

func (c componentDriverMinio) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, input *api.ComponentUpdateInput) error {
	return c.validateSetting(ctx, userCred, cluster, input.Minio)
}

func (c componentDriverMinio) GetCreateSettings(input *api.ComponentCreateInput) (*api.ComponentSettings, error) {
	if input.ComponentSettings.Namespace == "" {
		input.ComponentSettings.Namespace = MinioNamespace
	}
	return &input.ComponentSettings, nil
}

func (c componentDriverMinio) GetUpdateSettings(oldSetting *api.ComponentSettings, input *api.ComponentUpdateInput) (*api.ComponentSettings, error) {
	oldSetting.Minio = input.Minio
	return oldSetting, nil
}

func (c componentDriverMinio) DoEnable(cluster *SCluster, setting *api.ComponentSettings) error {
	return MinioComponentManager.CreateHelmResource(cluster, setting)
}

// TODO: refactor this deduplicated code
func (c componentDriverMinio) GetHelmValues(cluster *SCluster, setting *api.ComponentSettings) (map[string]interface{}, error) {
	return MinioComponentManager.GetHelmValues(cluster, setting)
}

func (c componentDriverMinio) DoDisable(cluster *SCluster, setting *api.ComponentSettings) error {
	return MinioComponentManager.DeleteHelmResource(cluster, setting)
}

func (c componentDriverMinio) DoUpdate(cluster *SCluster, setting *api.ComponentSettings) error {
	return MinioComponentManager.UpdateHelmResource(cluster, setting)
}

func (c componentDriverMinio) FetchStatus(cluster *SCluster, comp *SComponent, status *api.ComponentsStatus) error {
	if status.Minio == nil {
		status.Minio = new(api.ComponentStatus)
	}
	c.InitStatus(comp, status.Minio)
	return nil
}

func (m SMinioComponentManager) CreateHelmResource(cluster *SCluster, setting *api.ComponentSettings) error {
	vals, err := m.GetHelmValues(cluster, setting)
	if err != nil {
		return errors.Wrap(err, "get helm config values")
	}
	return m.HelmComponentManager.CreateHelmResource(cluster, vals)
}

func (m SMinioComponentManager) DeleteHelmResource(cluster *SCluster, setting *api.ComponentSettings) error {
	return m.HelmComponentManager.DeleteHelmResource(cluster)
}

func (m SMinioComponentManager) UpdateHelmResource(cluster *SCluster, setting *api.ComponentSettings) error {
	vals, err := m.GetHelmValues(cluster, setting)
	if err != nil {
		return errors.Wrap(err, "get helm config values")
	}
	return m.HelmComponentManager.UpdateHelmResource(cluster, vals)
}

func (m SMinioComponentManager) GetHelmValues(cluster *SCluster, setting *api.ComponentSettings) (map[string]interface{}, error) {
	imgRepo, err := cluster.GetImageRepository()
	if err != nil {
		return nil, errors.Wrapf(err, "get cluster %s repo", cluster.GetName())
	}
	repo := imgRepo.Url
	mi := func(name, tag string) components.Image {
		return components.Image{
			Repository: fmt.Sprintf("%s/%s", repo, name),
			Tag:        tag,
		}
	}

	input := setting.Minio
	conf := components.Minio{
		Image:              mi("minio", "RELEASE.2020-12-03T05-49-24Z"),
		McImage:            mi("mc", "RELEASE.2020-11-25T23-04-07Z"),
		HelmKubectlJqImage: mi("helm-kubectl-jq", "3.1.0"),
		Mode:               string(input.Mode),
		Replicas:           input.Replicas,
		DrivesPerNode:      input.DrivesPerNode,
		Zones:              input.Zones,
		AccessKey:          input.AccessKey,
		SecretKey:          input.SecretKey,
		MountPath:          input.MountPath,
		Persistence: components.MinioPersistence{
			Enabled:      true,
			StorageClass: input.Storage.ClassName,
			Size:         fmt.Sprintf("%dGi", input.Storage.SizeMB/1024),
		},
	}

	return components.GenerateHelmValues(conf), nil
}
