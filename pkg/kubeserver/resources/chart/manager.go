package chart

import (
	"yunion.io/x/kubecomps/pkg/kubeserver/resources"
)

var ChartManager *SChartManager

type SChartManager struct {
	*resources.SResourceBaseManager
}

func init() {
	ChartManager = &SChartManager{
		SResourceBaseManager: resources.NewResourceBaseManager("chart", "charts"),
	}
}
