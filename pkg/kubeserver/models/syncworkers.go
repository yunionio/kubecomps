package models

import (
	"yunion.io/x/onecloud/pkg/appsrv"
)

var (
	syncClusterWorker *appsrv.SWorkerManager
)

func init() {
	syncClusterWorker = appsrv.NewWorkerManager(
		"clusterSyncWorkerManager",
		4,
		2048,
		true,
	)
}

func RunSyncClusterTask(task appsrv.IWorkerTask) {
	syncClusterWorker.Run(task, nil, nil)
}
