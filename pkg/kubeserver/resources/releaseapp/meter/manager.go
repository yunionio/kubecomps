package meter

import (
	"yunion.io/x/kubecomps/pkg/kubeserver/resources/releaseapp"
)

var MeterAppManager *SMeterAppManager

type SMeterAppManager struct {
	*releaseapp.SReleaseAppManager
}

func init() {
	MeterAppManager = &SMeterAppManager{}

	MeterAppManager.SReleaseAppManager = releaseapp.NewReleaseAppManager(MeterAppManager, "app_meter", "app_meters")
}

func (man *SMeterAppManager) GetReleaseName() string {
	return "meter"
}

func (man *SMeterAppManager) GetChartName() string {
	return releaseapp.NewYunionRepoChartName("meter")
}

func (man *SMeterAppManager) GetConfigSets() releaseapp.ConfigSets {
	return releaseapp.GetYunionGlobalConfigSets()
}
