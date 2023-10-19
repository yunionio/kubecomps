package selfbuild

import (
	"fmt"
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/constants"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/clusters/addons"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/clusters/kubespray"
	"yunion.io/x/kubecomps/pkg/kubeserver/embed"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	onecloudcli "yunion.io/x/kubecomps/pkg/utils/onecloud/client"
	"yunion.io/x/kubecomps/pkg/utils/registry"
)

type sAwsDriver struct {
	*sBaseGuestDriver
}

func NewAwsDriver() ISelfBuildDriver {
	return &sAwsDriver{
		sBaseGuestDriver: newBaseGuestDriver(api.ProviderTypeAws),
	}
}

func (s *sAwsDriver) GetK8sVersions() []string {
	return []string{
		constants.K8S_VERSION_1_22_9,
	}
}

func (s *sAwsDriver) ChangeKubesprayVars(vars *kubespray.KubesprayVars) {
	// change aws download_url
	vars.KubeImageRepo = "k8s.gcr.io"
	vars.NodelocalDNSImageRepo = ""
	vars.DNSAutoscalerImageRepo = ""
	vars.NginxImageRepo = ""
	vars.CorednsImageIsNamespaced = true

	vars.KubeletDownloadUrl = ""
	vars.KubectlDownloadUrl = ""
	vars.KubeadmDownloadUrl = ""
	vars.EtcdDownloadUrl = ""
	vars.CNIDownloadUrl = ""
	vars.CalicoctlAlternateDownloadUrl = ""
	vars.CalicoctlDownloadUrl = ""
	vars.CalicoCRDsDownloadUrl = ""
	vars.CrictlDownloadUrl = ""
	vars.ContainerManager = "containerd"
	vars.ContainerdVersion = ""
	vars.EtcdDeploymentType = "host"
	vars.DockerVersion = kubespray.DockerVersion_20_10
	vars.DockerCliVersion = kubespray.DockerVersion_20_10
	vars.KubeNetworkPlugin = kubespray.NetworkPluginCNI
	vars.IngressNginxControllerImageRepo = ""
	vars.IngressNginxEnabled = false
	// vars.EnableNodelocalDNS = false
	// vars.OverrideSystemHostname = false

	// List of the preferred NodeAddressTypes to use for kubelet connections.
	// todo: 目前 aws instance 通过 InternalDNS 解析会失败，所以改成 InternalIP 优先
	vars.KubeletPreferredAddressTypes = "InternalIP,InternalDNS,Hostname,ExternalDNS,ExternalIP"

	// set cloud-provider to external
	vars.CloudProvider = kubespray.CloudProviderExternal
	vars.KubeKubeadmControllerExtraArgs = map[string]string{
		"cloud-provider": kubespray.CloudProviderExternal,
	}
	vars.KubeKubeadmControllerExtraArgs = map[string]string{
		"cloud-provider": kubespray.CloudProviderExternal,
	}
}

func (s *sAwsDriver) GetAddonsHelmCharts(cluster *models.SCluster, conf *api.ClusterAddonsManifestConfig) ([]*models.ClusterHelmChartInstallOption, error) {
	charts := []*models.ClusterHelmChartInstallOption{
		{
			EmbedChartName: embed.AWS_LOAD_BALANCER_CONTROLLER_1_5_5_TGZ,
			ReleaseName:    "aws-load-balancer-controller",
			Namespace:      "kube-system",
			Values: map[string]interface{}{
				"replicaCount":                1,
				"clusterName":                 cluster.GetName(),
				"enableServiceMutatorWebhook": false,
			},
		},
		{
			EmbedChartName: embed.AWS_EBS_CSI_DRIVER_2_22_0_TGZ,
			ReleaseName:    "aws-ebs-csi-driver",
			Namespace:      "kube-system",
			Values: map[string]interface{}{
				"storageClasses": []map[string]interface{}{
					{
						"name": "gp2",
						"annotations": map[string]interface{}{
							"storageclass.kubernetes.io/is-default-class": "false",
						},
						"reclaimPolicy":        "Delete",
						"volumeBindingMode":    "WaitForFirstConsumer",
						"allowVolumeExpansion": true,
						"parameters": map[string]interface{}{
							"type": "gp2",
						},
					},
				},
			},
		},
	}
	return charts, nil
}

func (s *sAwsDriver) GetAddonsManifest(cluster *models.SCluster, conf *api.ClusterAddonsManifestConfig) (string, error) {
	commonConf, err := GetCommonAddonsConfig(cluster)
	if err != nil {
		return "", err
	}
	commonConf.CloudProviderYunionConfig = nil
	commonConf.IngressControllerYunionConfig = nil
	commonConf.CSIYunionConfig = nil
	commonConf.MetricsPluginConfig = nil
	// commonConf.CSIRancherLocalPathConfig.Image = "registry.cn-beijing.aliyuncs.com/yunionio/local-path-provisioner:v0.0.24"
	commonConf.CSIRancherLocalPathConfig.Image = "rancher/local-path-provisioner:v0.0.24"
	commonConf.CSIRancherLocalPathConfig.HelperImage = "busybox:latest"

	// reg, err := cluster.GetImageRepository()
	// if err != nil {
	// 	return "", errors.Wrap(err, "get cluster image_repository")
	// }

	cniVersion := "v1.13.3"
	cniUrl := "registry.us-west-1.aliyuncs.com/yunion-dev"

	pluginConf := &addons.AwsVMPluginsConfig{
		YunionCommonPluginsConfig: commonConf,
		AwsVPCCNIConfig: &addons.AwsVPCCNIConfig{
			Image:     registry.MirrorImage(cniUrl, "amazon-k8s-cni", cniVersion, ""),
			InitImage: registry.MirrorImage(cniUrl, "amazon-k8s-cni-init", cniVersion, ""),
		},
		CloudProviderAwsConfig: &addons.CloudProviderAwsConfig{},
	}
	return pluginConf.GenerateYAML()
}

func (s *sAwsDriver) GetKubesprayHostname(info *onecloudcli.ServerSSHLoginInfo) (string, error) {
	// name format is: ip-<a-b-c-d>.<region>.compute.internal
	// e.g.: ip-10-1-22-51.ap-southeast-1.compute.internal
	// and us-east-1 region is:ip-<a-b-c-d>.ec2.internal
	ipFmt := strings.ReplaceAll(info.PrivateIP, ".", "-")
	regionId := info.CloudregionExternalId
	parts := strings.Split(regionId, "/")
	if len(parts) != 2 {
		return "", errors.Errorf("Invalid cloudregion external_id %q", regionId)
	}
	awsRegion := parts[1]
	if awsRegion == "us-east-1" {
		return fmt.Sprintf("ip-%s.ec2.internal", ipFmt), nil
	}
	return fmt.Sprintf("ip-%s.%s.compute.internal", ipFmt, awsRegion), nil
}
