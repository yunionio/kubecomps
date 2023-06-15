package chart

import (
	"fmt"
	"regexp"

	"yunion.io/x/log"
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
	query.Repo = repoObj.GetName()

	inputAllVersion := query.AllVersion
	if len(options.Options.ChartIgnores) != 0 {
		query.AllVersion = true
	}

	list, err := cli.SearchRepo(*query, query.Version) //, query.RepoUrl, query.Keyword)
	if err != nil {
		return nil, err
	}

	// execute ChartIngores
	list = man.executeChartIngores(query.Repo, inputAllVersion, list, options.Options.ChartIgnores)

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

func (man *SChartManager) executeChartIngores(repo string, allVersion bool, list []*api.ChartResult, ignores []string) []*api.ChartResult {
	if len(ignores) == 0 {
		return list
	}
	records := make(map[string]bool)
	ret := make([]*api.ChartResult, 0)
	log.Infof("Execute ignores exp: %v", ignores)
	for _, chart := range list {
		if man.shouldIngoreChart(repo, chart, ignores) {
			continue
		}
		if !allVersion {
			if ok := records[chart.Name]; !ok {
				records[chart.Name] = true
				ret = append(ret, chart)
			}
		} else {
			ret = append(ret, chart)
		}
	}
	return ret
}

func (man *SChartManager) shouldIngoreChart(repo string, chart *api.ChartResult, ignores []string) bool {
	chartKey := fmt.Sprintf("%s:%s:%s", repo, chart.Name, chart.Version)
	for _, iExp := range ignores {
		exp := regexp.MustCompile(iExp)
		if exp.Match([]byte(chartKey)) {
			log.Infof("Chart %q ignore by regexp %q", chartKey, iExp)
			return true
		}
	}
	return false
}
