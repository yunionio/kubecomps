package selfbuild

import (
	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/constants"
)

type sAliyunDriver struct {
	*sBaseGuestDriver
}

func NewAliyunDriver() ISelfBuildDriver {
	return &sAliyunDriver{
		sBaseGuestDriver: newBaseGuestDriver(api.ProviderTypeAliyun),
	}
}

func (s *sAliyunDriver) GetK8sVersions() []string {
	return []string{
		constants.K8S_VERSION_1_22_9,
		constants.K8S_VERSION_1_20_0,
	}
}
