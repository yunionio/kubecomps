package selfbuild

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/clusters/kubespray"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	"yunion.io/x/kubecomps/pkg/utils/onecloud/client"
)

type ISelfBuildDriver interface {
	GetProvider() api.ProviderType
	GetResourceType() api.ClusterResourceType
	GetK8sVersions() []string
	ChangeKubesprayVars(vars *kubespray.KubesprayVars)
	GetAddonsHelmCharts(cluster *models.SCluster, conf *api.ClusterAddonsManifestConfig) ([]*models.ClusterHelmChartInstallOption, error)
	GetAddonsManifest(cluster *models.SCluster, conf *api.ClusterAddonsManifestConfig) (string, error)
	GetKubesprayHostname(info *client.ServerSSHLoginInfo) (string, error)
	SetDefaultCreateData(ctx context.Context, cred mcclient.TokenCredential, id mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.ClusterCreateInput) error
}
