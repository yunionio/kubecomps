package models

import (
	"context"
	"strconv"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/templates/components"
	"yunion.io/x/kubecomps/pkg/utils/registry"
)

var (
	CephCSIComponentManager *SCephCSIComponentManager
)

const (
	CephCSIConfigMapName = "ceph-csi-config"
	CephCSINamespace     = "ceph-csi"
)

func init() {
	CephCSIComponentManager = NewCephCSIComponentManager()
	ComponentManager.RegisterDriver(newComponentDriverCephCSI())
}

type SCephCSIComponentManager struct {
	SComponentManager
	K8SComponentManager
}

type SCephCSIComponent struct {
	SComponent
}

func NewCephCSIComponentManager() *SCephCSIComponentManager {
	man := new(SCephCSIComponentManager)
	man.SComponentManager = *NewComponentManager(SCephCSIComponent{},
		"kubecomponentcephcsi",
		"kubecomponentcephcsis",
	)
	man.SetVirtualObject(man)
	return man
}

type componentDriverCephCSI struct {
	baseComponentDriver
}

func newComponentDriverCephCSI() IComponentDriver {
	return new(componentDriverCephCSI)
}

func (c componentDriverCephCSI) GetType() string {
	return api.ClusterComponentCephCSI
}

func (c componentDriverCephCSI) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, input *api.ComponentCreateInput) error {
	return c.validateSetting(input.CephCSI)
}

func (c componentDriverCephCSI) validateSetting(conf *api.ComponentSettingCephCSI) error {
	if conf == nil {
		return httperrors.NewNotEmptyError("cephCSI config")
	}
	if len(conf.Config) == 0 {
		return httperrors.NewNotEmptyError("cephCSI config is empty")
	}
	for _, conf := range conf.Config {
		if err := c.validateCreateConfig(conf); err != nil {
			return err
		}
	}
	return nil
}

func (c componentDriverCephCSI) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, input *api.ComponentUpdateInput) error {
	return c.validateSetting(input.CephCSI)
}

func (c componentDriverCephCSI) validateCreateConfig(conf api.ComponentCephCSIConfigCluster) error {
	if conf.ClsuterId == "" {
		return httperrors.NewNotEmptyError("cluster id is empty")
	}
	if len(conf.Monitors) == 0 {
		return httperrors.NewNotEmptyError("cluster %s monitors is empty", conf.ClsuterId)
	}
	for _, mon := range conf.Monitors {
		if err := c.validateMon(mon); err != nil {
			return err
		}
	}
	return nil
}

func (c componentDriverCephCSI) validateMon(mon string) error {
	parts := strings.Split(mon, ":")
	if len(parts) != 2 {
		return httperrors.NewInputParameterError("monitor format error, use 'ip:port'")
	}
	portStr := parts[1]
	if _, err := strconv.Atoi(portStr); err != nil {
		return httperrors.NewInputParameterError("monitor format port invalid: %v", err)
	}
	return nil
}

func (c componentDriverCephCSI) GetCreateSettings(input *api.ComponentCreateInput) (*api.ComponentSettings, error) {
	if input.ComponentSettings.Namespace == "" {
		input.ComponentSettings.Namespace = CephCSINamespace
	}
	return &input.ComponentSettings, nil
}

func (c componentDriverCephCSI) GetUpdateSettings(oldSetting *api.ComponentSettings, input *api.ComponentUpdateInput) (*api.ComponentSettings, error) {
	oldSetting.CephCSI = input.CephCSI
	return oldSetting, nil
}

func (c componentDriverCephCSI) DoEnable(cluster *SCluster, setting *api.ComponentSettings) error {
	return CephCSIComponentManager.ApplyK8sResource(cluster, setting)
}

func (c componentDriverCephCSI) DoDisable(cluster *SCluster, setting *api.ComponentSettings) error {
	return CephCSIComponentManager.DeleteK8sResource(cluster, setting)
}

func (c componentDriverCephCSI) GetHelmValues(cluster *SCluster, setting *api.ComponentSettings) (map[string]interface{}, error) {
	return nil, errors.Errorf("%s not managed by helm", c.GetType())
}

func (c componentDriverCephCSI) DoUpdate(cluster *SCluster, setting *api.ComponentSettings) error {
	if err := CephCSIComponentManager.DeleteK8sResource(cluster, setting); err != nil {
		return errors.Wrap(err, "delete ceph csi resource when update")
	}
	return CephCSIComponentManager.ApplyK8sResource(cluster, setting)
}

func (m componentDriverCephCSI) FetchStatus(cluster *SCluster, component *SComponent, status *api.ComponentsStatus) error {
	if status.CephCSI == nil {
		status.CephCSI = new(api.ComponentStatusCephCSI)
	}
	m.InitStatus(component, &status.CephCSI.ComponentStatus)
	return nil
}

type CephCSIClusterConfig struct {
	ClusterId string   `json:"clusterID"`
	Monitors  []string `json:"monitors"`
}

func newCephCSIClusterConfigBySettings(settings *api.ComponentSettingCephCSI) []CephCSIClusterConfig {
	ret := make([]CephCSIClusterConfig, 0)
	for _, conf := range settings.Config {
		ret = append(ret, CephCSIClusterConfig{
			ClusterId: conf.ClsuterId,
			Monitors:  conf.Monitors,
		})
	}
	return ret
}

func fetchCephCSIClusterConfig(cluster *SCluster, namespace string) (*api.ComponentSettingCephCSI, error) {
	cli, err := cluster.GetK8sClient()
	if err != nil {
		return nil, err
	}
	configMap, err := cli.CoreV1().ConfigMaps(namespace).Get(context.Background(), CephCSIConfigMapName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	config := make([]CephCSIClusterConfig, 0)
	data := configMap.Data["config.json"]
	obj, err := jsonutils.ParseString(data)
	if err != nil {
		return nil, errors.Wrapf(err, "parse configmap %s data", CephCSIConfigMapName)
	}
	if err := obj.Unmarshal(&config); err != nil {
		return nil, errors.Wrapf(err, "")
	}
	ret := new(api.ComponentSettingCephCSI)
	ret.Config = make([]api.ComponentCephCSIConfigCluster, 0)
	for _, c := range config {
		ret.Config = append(ret.Config, api.ComponentCephCSIConfigCluster{
			ClsuterId: c.ClusterId,
			Monitors:  c.Monitors,
		})
	}
	return ret, nil
}

func (m *SCephCSIComponentManager) getK8sResourceManifest(cluster *SCluster, setting *api.ComponentSettings) (string, error) {
	imgRepo, err := cluster.GetImageRepository()
	if err != nil {
		return "", errors.Wrapf(err, "get cluster %s repo", cluster.GetName())
	}
	configJson := jsonutils.Marshal(newCephCSIClusterConfigBySettings(setting.CephCSI)).String()
	repo := imgRepo.Url
	mi := registry.MirrorImage
	conf := components.CephCSIRBDConfig{
		Namespace:        setting.Namespace,
		ConfigMapName:    CephCSIConfigMapName,
		ConfigJSON:       configJson,
		AttacherImage:    mi(repo, "csi-attacher", "v2.1.0", ""),
		ProvisionerImage: mi(repo, "csi-provisioner", "v1.4.0", ""),
		SnapshotterImage: mi(repo, "csi-snapshotter", "v1.2.2", ""),
		CephCSIImage:     mi(repo, "cephcsi", "v2.0-canary", ""),
		RegistrarImage:   mi(repo, "csi-node-driver-registrar", "v1.2.0", ""),
		ResizerImage:     mi(repo, "csi-resizer", "v0.4.0", ""),
	}
	manifest, err := conf.GenerateYAML()
	if err != nil {
		return "", errors.Wrap(err, "ceph csi get manifest")
	}
	return manifest, nil
}

func (m *SCephCSIComponentManager) ApplyK8sResource(cluster *SCluster, setting *api.ComponentSettings) error {
	if err := m.EnsureNamespace(cluster, setting.Namespace); err != nil {
		return errors.Wrapf(err, "ceph csi ensure namespace %q", setting.Namespace)
	}
	manifest, err := m.getK8sResourceManifest(cluster, setting)
	if err != nil {
		return err
	}
	return m.KubectlApply(cluster, manifest)
}

func (m *SCephCSIComponentManager) DeleteK8sResource(cluster *SCluster, setting *api.ComponentSettings) error {
	manifest, err := m.getK8sResourceManifest(cluster, setting)
	if err != nil {
		return err
	}
	return m.KubectlDelete(cluster, manifest)
}
