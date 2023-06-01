package initial

import (
	"time"

	// "k8s.io/apimachinery/pkg/util/wait"

	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"

	"yunion.io/x/kubecomps/pkg/kubeserver/client"
	"yunion.io/x/kubecomps/pkg/kubeserver/controllers/helm"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	"yunion.io/x/kubecomps/pkg/kubeserver/models/manager"
	"yunion.io/x/kubecomps/pkg/kubeserver/options"

	_ "yunion.io/x/kubecomps/pkg/kubeserver/drivers/clusters"
	_ "yunion.io/x/kubecomps/pkg/kubeserver/drivers/machines"
	_ "yunion.io/x/kubecomps/pkg/kubeserver/tasks"
)

func InitClient(cron *cronman.SCronJobManager) {
	// go wait.Forever(client.BuildApiserverClient, 5*time.Second)
	client.InitClustersManager(manager.ClusterManager())

	cron.AddJobAtIntervalsWithStartRun("StartKubeClusterHealthCheck", 5*time.Minute, models.ClusterManager.ClusterHealthCheckTask, true)
	cron.AddJobAtIntervalsWithStartRun("StartKubeClusterAutoSyncTask", 30*time.Minute, models.ClusterManager.StartAutoSyncTask, true)
	if options.Options.RunningMode == options.RUNNING_MODE_K8S {
		cron.AddJobAtIntervalsWithStartRun("StartSyncSystemGrafanaDashboard", 1*time.Minute, models.MonitorComponentManager.SyncSystemGrafanaDashboard, true)
	}

	syncPeriod := options.Options.RepoRefreshDuration
	cron.AddJobAtIntervalsWithStartRun("StartHelmRepoSyncTask", time.Duration(syncPeriod)*time.Minute,
		helm.Start, true)
}
