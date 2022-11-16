package clusters

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	perrors "yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/clusters/addons"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	"yunion.io/x/kubecomps/pkg/kubeserver/options"
	"yunion.io/x/kubecomps/pkg/utils/registry"
)

func GetAddonYunionAuthConfig(cluster *models.SCluster) (addons.YunionAuthConfig, error) {
	o := options.Options
	s, _ := models.ClusterManager.GetSession()
	authConfig := addons.YunionAuthConfig{
		AuthUrl:       o.AuthURL,
		AdminUser:     o.AdminUser,
		AdminPassword: o.AdminPassword,
		AdminProject:  o.AdminProject,
		Region:        o.Region,
		Cluster:       cluster.GetName(),
		InstanceType:  cluster.ResourceType,
	}
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString("public"), "interface")
	params.Add(jsonutils.NewString(o.Region), "region")
	params.Add(jsonutils.NewString("keystone"), "service")
	ret, err := identity.EndpointsV3.List(s, params)
	if err != nil {
		return authConfig, err
	}
	if len(ret.Data) == 0 {
		return authConfig, perrors.Error("Not found public keystone endpoint")
	}
	authUrl, err := ret.Data[0].GetString("url")
	if err != nil {
		return authConfig, perrors.Wrap(err, "Get public keystone endpoint url")
	}
	authConfig.AuthUrl = authUrl
	return authConfig, nil
}

func GetCommonAddonsConfig(cluster *models.SCluster) (*addons.YunionCommonPluginsConfig, error) {
	authConfig, err := GetAddonYunionAuthConfig(cluster)
	if err != nil {
		return nil, err
	}
	reg, err := cluster.GetImageRepository()
	if err != nil {
		return nil, err
	}

	commonConf := &addons.YunionCommonPluginsConfig{
		/* MetricsPluginConfig: &addons.MetricsPluginConfig{
			MetricsServerImage: registry.MirrorImage(reg.Url, "metrics-server-amd64", "v0.3.1", ""),
		}, */
		/*
		 * HelmPluginConfig: &addons.HelmPluginConfig{
		 *     TillerImage: registry.MirrorImage(reg.Url, "tiller", "v2.11.0", ""),
		 * },
		 */
		CloudProviderYunionConfig: &addons.CloudProviderYunionConfig{
			YunionAuthConfig:   authConfig,
			CloudProviderImage: registry.MirrorImage(reg.Url, "yunion-cloud-controller-manager", "v2.10.0", ""),
		},
		/* CSIYunionConfig: &addons.CSIYunionConfig{
			YunionAuthConfig: authConfig,
			AttacherImage:    registry.MirrorImage(reg.Url, "csi-attacher", "v1.0.1", ""),
			ProvisionerImage: registry.MirrorImage(reg.Url, "csi-provisioner", "v1.0.1", ""),
			RegistrarImage:   registry.MirrorImage(reg.Url, "csi-node-driver-registrar", "v1.1.0", ""),
			PluginImage:      registry.MirrorImage(reg.Url, "yunion-csi-plugin", "v2.10.0", ""),
			Base64Config:     authConfig.ToJSONBase64String(),
		}, */
		// IngressControllerYunionConfig: &addons.IngressControllerYunionConfig{
		// 	YunionAuthConfig: authConfig,
		// 	Image:            registry.MirrorImage(reg.Url, "yunion-ingress-controller", "v2.10.0", ""),
		// },
		/* CSIRancherLocalPathConfig: &addons.CSIRancherLocalPathConfig{
			Image:       registry.MirrorImage(reg.Url, "local-path-provisioner", "v0.0.11", ""),
			HelperImage: registry.MirrorImage(reg.Url, "busybox", "1.28.0-glibc", ""),
		}, */
	}

	return commonConf, nil
}
