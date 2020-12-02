package release

import (
	"helm.sh/helm/v3/pkg/release"

	"yunion.io/x/log"

	api "yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/helm"
	"yunion.io/x/kubecomps/pkg/kubeserver/resources/common"
)

func (man *SReleaseManager) AllowUpdateItem(req *common.Request, id string) bool {
	return man.AllowDeleteItem(req, id)
}

func (man *SReleaseManager) Update(req *common.Request, id string) (interface{}, error) {
	input := &api.ReleaseUpdateInput{}
	if err := req.DataUnmarshal(input); err != nil {
		return nil, err
	}
	cli, err := req.GetHelmClient(input.Namespace)
	if err != nil {
		return nil, err
	}
	return ReleaseUpgrade(cli.Release(), input)
}

func ReleaseUpgrade(helmclient helm.IRelease, opt *api.ReleaseUpdateInput) (*release.Release, error) {
	log.Infof("Upgrade repo=%q, chart=%q, release='%s/%s'", opt.Repo, opt.ChartName, opt.Namespace, opt.ReleaseName)
	return helmclient.Update(opt)
}
