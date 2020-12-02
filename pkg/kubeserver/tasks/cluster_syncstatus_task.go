package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	"yunion.io/x/kubecomps/pkg/utils/logclient"
)

func init() {
	taskman.RegisterTask(ClusterSyncstatusTask{})
}

type ClusterSyncstatusTask struct {
	taskman.STask
}

func (t *ClusterSyncstatusTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*models.SCluster)
	mCnt, err := cluster.GetMachinesCount()
	if err != nil {
		t.onError(ctx, cluster, err.Error())
		return
	}
	if mCnt == 0 && cluster.GetDriver().NeedCreateMachines() {
		cluster.SetStatus(t.UserCred, api.ClusterStatusInit, "")
		t.SetStageComplete(ctx, nil)
		return
	}

	t.SetStage("OnSyncStatus", nil)
	taskman.LocalTaskRun(t, func() (jsonutils.JSONObject, error) {
		k8sCli, err := cluster.GetK8sClient()
		if err != nil {
			return nil, err
		}
		info, err := k8sCli.Discovery().ServerVersion()
		if err != nil {
			return nil, err
		}
		log.Infof("Get %s cluster k8s version: %#v", cluster.GetName(), info)
		if err := cluster.SetStatus(t.UserCred, api.ClusterStatusRunning, ""); err != nil {
			return nil, errors.Wrap(err, "set status to running")
		}
		cluster.SetK8sVersion(ctx, info.String())
		return nil, nil
	})
}

func (t *ClusterSyncstatusTask) OnSyncStatus(ctx context.Context, cluster *models.SCluster, data jsonutils.JSONObject) {
	logclient.LogWithStartable(t, cluster, logclient.ActionClusterSyncStatus, nil, t.UserCred, true)
	t.SetStageComplete(ctx, nil)
}

func (t *ClusterSyncstatusTask) OnSyncStatusFailed(ctx context.Context, cluster *models.SCluster, data jsonutils.JSONObject) {
	t.onError(ctx, cluster, data.String())
}

func (t *ClusterSyncstatusTask) onError(ctx context.Context, cluster db.IStandaloneModel, err string) {
	t.SetFailed(ctx, cluster, jsonutils.NewString(err))
	logclient.LogWithStartable(t, cluster, logclient.ActionClusterSyncStatus, err, t.UserCred, false)
}

func (t *ClusterSyncstatusTask) SetFailed(ctx context.Context, obj db.IStandaloneModel, reason jsonutils.JSONObject) {
	cluster := obj.(*models.SCluster)
	cluster.SetStatus(t.UserCred, api.ClusterStatusUnknown, "")
	t.STask.SetStageFailed(ctx, reason)
}
