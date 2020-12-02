package chart

import (
	"path/filepath"

	"encoding/base64"

	"helm.sh/helm/v3/pkg/chart"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/helm"
	"yunion.io/x/kubecomps/pkg/kubeserver/options"
)

func (man *SChartManager) Show(repoName, chartName, version string) (interface{}, error) {
	return man.GetDetails(repoName, chartName, version)
}

func GetChartRawFiles(chObj *chart.Chart) []*chart.File {
	files := make([]*chart.File, len(chObj.Raw))
	for idx, rf := range chObj.Raw {
		files[idx] = &chart.File{
			Name: filepath.Join(chObj.Name(), rf.Name),
			Data: rf.Data,
		}
	}
	return files
}

func (man *SChartManager) GetDetails(repoName, chartName, version string) (*api.ChartDetail, error) {
	chObj, err := helm.NewChartClient(options.Options.HelmDataDir).Show(repoName, chartName, version)
	if err != nil {
		return nil, err
	}
	readmeStr := ""
	readmeFile := helm.FindChartReadme(chObj)
	if readmeFile != nil {
		readmeStr = base64.StdEncoding.EncodeToString(readmeFile.Data)
	}
	return &api.ChartDetail{
		Repo:   repoName,
		Name:   chObj.Name(),
		Chart:  chObj,
		Readme: readmeStr,
		Files:  GetChartRawFiles(chObj),
	}, nil
}
