package imported

import "yunion.io/x/kubecomps/pkg/kubeserver/api"

type CloudQcloudK8s struct {
	*cloudK8sBaseDriver
}

func NewCloudQcloudK8s() *CloudQcloudK8s {
	return &CloudQcloudK8s{
		cloudK8sBaseDriver: newCloudK8sBaseDriver(api.ProviderTypeQcloud),
	}
}
