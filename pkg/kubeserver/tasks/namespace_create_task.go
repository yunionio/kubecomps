package tasks

import (
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
)

func init() {
	taskman.RegisterTask(NamespaceCreateTask{})
}

type NamespaceCreateTask struct {
	ClusterResourceCreateTask
}
