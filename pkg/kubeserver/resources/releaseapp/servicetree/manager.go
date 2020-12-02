package servicetree

import (
	"yunion.io/x/kubecomps/pkg/kubeserver/resources/releaseapp"
)

var ServicetreeAppManager *SServicetreeAppManager

type SServicetreeAppManager struct {
	*releaseapp.SReleaseAppManager
}

func init() {
	ServicetreeAppManager = &SServicetreeAppManager{}

	ServicetreeAppManager.SReleaseAppManager = releaseapp.NewReleaseAppManager(ServicetreeAppManager, "app_servicetree", "app_servicetrees")
}

func (man *SServicetreeAppManager) GetConfigSets() releaseapp.ConfigSets {
	globalSets := releaseapp.GetYunionInfluxdbGlobalConfigSets()
	conf := map[string]string{
		releaseapp.INFLUXDB_DB_CONF_KEY: "telegraf",
		"monitor-stream.kafka.replicas": "1",
		`monitor-stream.kafka.configurationOverrides.offsets\.topic\.replication\.factor`: "1",
		"monitor-stream.kafka.zookeeper.replicaCount":                                     "1",
		"monitor-stream.kairosdb.cassandra.config.cluster_size":                           "1",
		"monitor-stream.kairosdb.cassandra.config.seed_size":                              "1",
	}
	return globalSets.Add(conf)
}

func (man *SServicetreeAppManager) GetReleaseName() string {
	return "monitor"
}

func (man *SServicetreeAppManager) GetChartName() string {
	return releaseapp.NewYunionRepoChartName("monitor-stack")
}
