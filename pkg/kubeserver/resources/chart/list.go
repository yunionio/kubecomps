package chart

import (
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/helm"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	"yunion.io/x/kubecomps/pkg/kubeserver/options"
	"yunion.io/x/kubecomps/pkg/kubeserver/resources/common"
	"yunion.io/x/kubecomps/pkg/kubeserver/resources/dataselect"
)

type Chart struct {
	*api.ChartResult
}

func ToChart(repo *models.SRepo, ret *api.ChartResult) Chart {
	ret.Type = repo.GetType()
	return Chart{ret}
}

type ChartList struct {
	*dataselect.ListMeta
	Charts []Chart
	Repo   *models.SRepo
}

func (l *ChartList) GetResponseData() interface{} {
	return l.Charts
}

func (l *ChartList) Append(obj interface{}) {
	l.Charts = append(l.Charts, ToChart(l.Repo, obj.(*api.ChartResult)))
}

func (man *SChartManager) List(userCred mcclient.TokenCredential, query *api.ChartListInput, dsQuery *dataselect.DataSelectQuery) (common.ListResource, error) {
	cli := helm.NewChartClient(options.Options.HelmDataDir)
	repo := query.Repo
	if repo == "" {
		return nil, httperrors.NewNotEmptyError("repo must provided")
	}
	repoObj, err := models.RepoManager.FetchByIdOrName(userCred, repo)
	if err != nil {
		return nil, err
	}
	list, err := cli.SearchRepo(*query, query.Version) //, query.RepoUrl, query.Keyword)
	if err != nil {
		return nil, err
	}
	chartList := &ChartList{
		ListMeta: dataselect.NewListMeta(),
		Charts:   make([]Chart, 0),
		Repo:     repoObj.(*models.SRepo),
	}
	err = dataselect.ToResourceList(
		chartList,
		list,
		dataselect.NewChartDataCell,
		dsQuery,
	)
	return chartList, err
}
