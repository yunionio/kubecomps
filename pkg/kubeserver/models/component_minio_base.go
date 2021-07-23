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

type SMinioBaseComponentManager struct {
	SComponentManager
	HelmComponentManager
}

type SMinioBaseComponent struct {
	SComponent
}

func NewMinioBaseComponentManager(
	key, keyPlural string,
	releaseName string,
	namespace string,
	dt interface{},
) *SMinioBaseComponentManager {
	man := new(SMinioBaseComponentManager)
	man.SComponentManager = *NewComponentManager(dt,
		key,
		keyPlural,
	)
	man.HelmComponentManager = *NewHelmComponentManager(namespace, releaseName, embed.MINIO_8_0_6_TGZ)
	return man
}

func (m SMinioBaseComponentManager) GetHelmManager() HelmComponentManager {
	return m.HelmComponentManager
}

type componentDriverMinioBase struct {
	baseComponentDriver
	driverType string
}

func newComponentDriverMinioBase(typ string) *componentDriverMinioBase {
	return &componentDriverMinioBase{
		driverType: typ,
	}
}

func (b componentDriverMinioBase) GetType() string {
	return b.driverType
}

func (c componentDriverMinioBase) validateSetting(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, conf *api.ComponentSettingMinio) error {
	if conf == nil {
		return httperrors.NewNotEmptyError("%s config is empty", c.GetType())
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

func (c componentDriverMinioBase) getCreateSettings(input *api.ComponentCreateInput, namespace string) (*api.ComponentSettings, error) {
	if input.ComponentSettings.Namespace == "" {
		input.ComponentSettings.Namespace = namespace
	}
	return &input.ComponentSettings, nil
}

func (c componentDriverMinioBase) fetchStatus(cluster *SCluster, comp *SComponent, status *api.ComponentStatus) error {
	c.InitStatus(comp, status)
	return nil
}

type IMinioComponentManager interface {
	GetHelmManager() HelmComponentManager
	CreateHelmResource(cluster *SCluster, setting *api.ComponentSettings) error
	GetHelmValues(cluster *SCluster, setting *api.ComponentSettings) (map[string]interface{}, error)
	UpdateHelmResource(cluster *SCluster, setting *api.ComponentSettings) error
	DeleteHelmResource(cluster *SCluster, setting *api.ComponentSettings) error
}

func DoEnableMinio(man IMinioComponentManager, cluster *SCluster, setting *api.ComponentSettings) error {
	return man.CreateHelmResource(cluster, setting)
}

func DoUpdateMinio(man IMinioComponentManager, cluster *SCluster, setting *api.ComponentSettings) error {
	return man.UpdateHelmResource(cluster, setting)
}

func DoDisableMinio(man IMinioComponentManager, cluster *SCluster, setting *api.ComponentSettings) error {
	return man.DeleteHelmResource(cluster, setting)
}

func CreateMinioHelmResource(man HelmComponentManager, cluster *SCluster, input *api.ComponentSettingMinio) error {
	vals, err := GetMinioHelmValues(cluster, input)
	if err != nil {
		return errors.Wrap(err, "get helm config values")
	}
	return man.CreateHelmResource(cluster, vals)
}

func UpdateMinioHelmResource(man HelmComponentManager, cluster *SCluster, input *api.ComponentSettingMinio) error {
	vals, err := GetMinioHelmValues(cluster, input)
	if err != nil {
		return errors.Wrap(err, "get helm config values")
	}
	return man.UpdateHelmResource(cluster, vals)
}

func DeleteMinioHelmResource(man HelmComponentManager, cluster *SCluster) error {
	return man.DeleteHelmResource(cluster)
}

func GetMinioHelmValues(cluster *SCluster, input *api.ComponentSettingMinio) (map[string]interface{}, error) {
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

	conf := components.Minio{
		Image:              mi("minio", "RELEASE.2021-06-17T00-10-46Z"),
		McImage:            mi("mc", "RELEASE.2021-06-13T17-48-22Z"),
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

	if cluster.IsSystemCluster() {
		conf.CommonConfig = getSystemComponentCommonConfig(true)
	}

	return components.GenerateHelmValues(conf), nil
}
