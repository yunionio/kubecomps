package clusters

import (
	"yunion.io/x/kubecomps/pkg/kubeserver/api"
)

func init() {
	registerSelfBuildClusterDriver(api.ProviderTypeAliyun)
}
