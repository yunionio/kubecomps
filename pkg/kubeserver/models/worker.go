package models

import (
	"yunion.io/x/onecloud/pkg/appsrv"
)

var taskWorkMan *appsrv.SWorkerManager

func init() {
	taskWorkMan = appsrv.NewWorkerManager("TaskWorkerManager", 4, 100, true)
}

func TaskManager() *appsrv.SWorkerManager {
	return taskWorkMan
}
