package release

import (
	"yunion.io/x/kubecomps/pkg/kubeserver/resources"
)

var ReleaseManager *SReleaseManager

type SReleaseManager struct {
	*resources.SNamespaceResourceManager
}

func init() {
	ReleaseManager = &SReleaseManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("release", "releases"),
	}
}
