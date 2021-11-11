package imported

import "yunion.io/x/kubecomps/pkg/kubeserver/api"

type CloudAzureK8s struct {
	*cloudK8sBaseDriver
}

func NewCloudAzureK8s() *CloudAzureK8s {
	return &CloudAzureK8s{
		cloudK8sBaseDriver: newCloudK8sBaseDriver(api.ProviderTypeAzure),
	}
}
