package notify

import (
	"fmt"
	"net/url"

	"yunion.io/x/log"

	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	"yunion.io/x/kubecomps/pkg/kubeserver/resources/releaseapp"
)

var NotifyAppManager *SNotifyAppManager

type SNotifyAppManager struct {
	*releaseapp.SReleaseAppManager
}

func init() {
	NotifyAppManager = &SNotifyAppManager{}

	NotifyAppManager.SReleaseAppManager = releaseapp.NewReleaseAppManager(NotifyAppManager, "app_notify", "app_notifies")
}

func (man *SNotifyAppManager) GetChartName() string {
	return releaseapp.NewYunionRepoChartName("notify")
}

func (man *SNotifyAppManager) GetReleaseName() string {
	return "notify"
}

func (man *SNotifyAppManager) GetConfigSets() releaseapp.ConfigSets {
	globalSets := releaseapp.GetYunionGlobalConfigSets()
	return globalSets.Add(map[string]string{
		"verification.template.email.title": "Yunion-email-activate-title",
		"verification.template.email.url":   man.getVerificationUrl(),
	})
}

func (man *SNotifyAppManager) getVerificationUrl() string {
	session, err := models.GetAdminSession()
	if err != nil {
		log.Errorf("Get admin session error: %v", err)
		return ""
	}
	influxdbUrl, _ := session.GetServiceURL("influxdb", "publicURL")
	u, err := url.Parse(influxdbUrl)
	if err != nil {
		log.Errorf("Parse publicURL %q error: %v", influxdbUrl, err)
		return ""
	}
	return fmt.Sprintf("https://%s/resource/email-verification/id/{0}/token/{1}", u.Hostname())
}
