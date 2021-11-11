package imported

import "yunion.io/x/kubecomps/pkg/kubeserver/api"

type CloudAliyunK8s struct {
	*cloudK8sBaseDriver
}

func NewCloudAliyunK8s() *CloudAliyunK8s {
	return &CloudAliyunK8s{
		cloudK8sBaseDriver: newCloudK8sBaseDriver(api.ProviderTypeAliyun),
	}
}
