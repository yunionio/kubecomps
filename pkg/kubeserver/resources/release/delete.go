package release

import (
	"yunion.io/x/kubecomps/pkg/kubeserver/helm"
	"yunion.io/x/kubecomps/pkg/kubeserver/resources/common"
)

func (man *SReleaseManager) IsRawResource() bool {
	return false
}

func (man *SReleaseManager) AllowDeleteItem(req *common.Request, id string) bool {
	return man.SNamespaceResourceManager.AllowDeleteItem(req, id)
}

func (man *SReleaseManager) Delete(req *common.Request, id string) error {
	cli, err := req.GetHelmClient(req.GetDefaultNamespace())
	if err != nil {
		return err
	}
	return ReleaseDelete(cli.Release(), id)
}

func ReleaseDelete(helmclient helm.IRelease, releaseName string) error {
	_, err := helmclient.UnInstall().Run(releaseName)
	return err
}
