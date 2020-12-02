package releaseapp

import (
	"fmt"

	"yunion.io/x/log"

	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	"yunion.io/x/kubecomps/pkg/kubeserver/options"
	"yunion.io/x/kubecomps/pkg/kubeserver/resources"
	"yunion.io/x/kubecomps/pkg/kubeserver/resources/release"
)

const (
	INFLUXDB_DB_CONF_KEY = "global.influxdb.database"
	YUNION_REPO_NAME     = "default"
)

type IReleaseAppHooker interface {
	GetReleaseName() string
	GetChartName() string
	GetConfigSets() ConfigSets
}

type SReleaseAppManager struct {
	*release.SReleaseManager
	hooker IReleaseAppHooker
}

func NewReleaseAppManager(hooker IReleaseAppHooker, keyword, keywordPlural string) *SReleaseAppManager {
	return &SReleaseAppManager{
		SReleaseManager: &release.SReleaseManager{
			SNamespaceResourceManager: resources.NewNamespaceResourceManager(keyword, keywordPlural),
		},
		hooker: hooker,
	}
}

type ConfigSets map[string]string

func NewConfigSets(conf map[string]string) ConfigSets {
	return ConfigSets(conf)
}

func (s ConfigSets) ToSets() []string {
	ret := make([]string, 0)
	for k, v := range s {
		ret = append(ret, fmt.Sprintf("%s=%s", k, v))
	}
	return ret
}

func (s ConfigSets) add(key, val string) ConfigSets {
	s[key] = val
	return s
}

func (s ConfigSets) Add(conf ConfigSets) ConfigSets {
	for k, v := range conf {
		s = s.add(k, v)
	}
	return s
}

func GetYunionGlobalConfigSets() ConfigSets {
	o := options.Options
	return map[string]string{
		"global.yunion.auth.url":      o.AuthURL,
		"global.yunion.auth.domain":   "Default",
		"global.yunion.auth.username": o.AdminUser,
		"global.yunion.auth.password": o.AdminPassword,
		"global.yunion.auth.project":  o.AdminProject,
		"global.yunion.auth.region":   o.Region,
	}
}

func GetYunionInfluxdbGlobalConfigSets() ConfigSets {
	conf := make(map[string]string)
	session, err := models.GetAdminSession()
	if err != nil {
		log.Errorf("Get admin session error: %v", err)
		return conf
	}
	influxdbUrl, _ := session.GetServiceURL("influxdb", "internalURL")
	conf = map[string]string{
		"global.influxdb.url": influxdbUrl,
	}
	return GetYunionGlobalConfigSets().Add(conf)
}

func NewYunionRepoChartName(chartName string) string {
	return fmt.Sprintf("%s/%s", YUNION_REPO_NAME, chartName)
}
