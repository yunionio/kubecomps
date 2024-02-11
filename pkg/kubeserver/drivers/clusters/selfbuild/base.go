package selfbuild

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/constants"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/clusters/addons"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/clusters/kubespray"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	onecloudcli "yunion.io/x/kubecomps/pkg/utils/onecloud/client"
)

type sBaseDriver struct {
	providerType api.ProviderType
	resourceType api.ClusterResourceType
}

func newBaseDriver(pt api.ProviderType, rt api.ClusterResourceType) ISelfBuildDriver {
	return &sBaseDriver{
		providerType: pt,
		resourceType: rt,
	}
}

func (s sBaseDriver) GetProvider() api.ProviderType {
	return s.providerType
}

func (s sBaseDriver) GetResourceType() api.ClusterResourceType {
	return s.resourceType
}

func (s sBaseDriver) GetK8sVersions() []string {
	return []string{
		constants.K8S_VERSION_1_17_0,
		constants.K8S_VERSION_1_20_0,
		constants.K8S_VERSION_1_22_9,
	}
}

func (s sBaseDriver) ChangeKubesprayVars(vars *kubespray.KubesprayVars) {
	return
}

func (s sBaseDriver) GetAddonsHelmCharts(cluster *models.SCluster, conf *api.ClusterAddonsManifestConfig) ([]*models.ClusterHelmChartInstallOption, error) {
	return nil, nil
}

func (s sBaseDriver) SetDefaultCreateData(ctx context.Context, cred mcclient.TokenCredential, id mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.ClusterCreateInput) error {
	if input.AddonsConfig == nil {
		input.AddonsConfig = new(api.ClusterAddonsManifestConfig)
	}
	if input.AddonsConfig.Ingress.EnableNginx == nil {
		defaultTrue := true
		input.AddonsConfig.Ingress.EnableNginx = &defaultTrue
	}
	return nil
}

func (s sBaseDriver) GetAddonsManifest(cluster *models.SCluster, conf *api.ClusterAddonsManifestConfig) (string, error) {
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

func (s *sBaseDriver) GetKubesprayHostname(info *onecloudcli.ServerSSHLoginInfo) (string, error) {
	return info.Hostname, nil
}

type sBaseGuestDriver struct {
	*sBaseDriver
}

func newBaseGuestDriver(pt api.ProviderType) *sBaseGuestDriver {
	return &sBaseGuestDriver{
		sBaseDriver: newBaseDriver(pt, api.ClusterResourceTypeGuest).(*sBaseDriver),
	}
}
