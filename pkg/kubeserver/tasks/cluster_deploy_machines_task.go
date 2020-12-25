package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	"yunion.io/x/kubecomps/pkg/kubeserver/models/manager"
	"yunion.io/x/kubecomps/pkg/utils/logclient"
)

// ClusterDeployMachinesTask run ansible deploy machines as kubernetes nodes
type ClusterDeployMachinesTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ClusterDeployMachinesTask{})
}

func (t *ClusterDeployMachinesTask) getAddMachines() (api.ClusterDeployAction, []manager.IMachine, error) {
	action, err := models.GetDataDeployAction(t.Params)
	if err != nil {
		return "", nil, errors.Wrap(err, "get deploy action")
	}
	msIds, err := models.GetDataDeployMachineIds(t.Params)
	if err != nil {
		return "", nil, errors.Wrap(err, "get deploy machine ids")
	}
	ms := make([]manager.IMachine, 0)
	for _, id := range msIds {
		m, err := models.MachineManager.FetchMachineById(id)
		if err != nil {
			return "", nil, err
		}
		ms = append(ms, m)
	}
	return action, ms, nil
}

func (t *ClusterDeployMachinesTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*models.SCluster)
	action, ms, err := t.getAddMachines()
	if err != nil {
		t.SetFailed(ctx, cluster, jsonutils.NewString(err.Error()))
		return
	}
	t.SetStage("OnDeployMachines", nil)
	if err := cluster.GetDriver().RequestDeployMachines(ctx, t.UserCred, cluster, action, ms, t); err != nil {
		t.SetFailed(ctx, cluster, jsonutils.NewString(err.Error()))
		return
	}
}

func (t *ClusterDeployMachinesTask) OnDeployMachines(ctx context.Context, cluster *models.SCluster, data jsonutils.JSONObject) {
	t.SetStage("OnSyncStatusComplete", nil)
	cluster.StartSyncStatus(ctx, t.UserCred, t.GetTaskId())
}

func (t *ClusterDeployMachinesTask) OnDeployMachinesFailed(ctx context.Context, cluster *models.SCluster, data jsonutils.JSONObject) {
	t.SetFailed(ctx, cluster, data)
}

func (t *ClusterDeployMachinesTask) OnSyncStatusComplete(ctx context.Context, cluster *models.SCluster, data jsonutils.JSONObject) {
	t.SetStage("OnSyncComplete", nil)
	if err := cluster.StartSyncTask(ctx, t.UserCred, nil, t.GetTaskId()); err != nil {
		t.SetFailed(ctx, cluster, jsonutils.NewString(err.Error()))
	}
}

func (t *ClusterDeployMachinesTask) OnSyncStatusCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	t.SetFailed(ctx, obj, data)
}

func (t *ClusterDeployMachinesTask) OnSyncComplete(ctx context.Context, cluster *models.SCluster, data jsonutils.JSONObject) {
	logclient.LogWithStartable(t, cluster, logclient.ActionClusterDeploy, nil, t.UserCred, true)
	t.SetStageComplete(ctx, nil)
}

func (t *ClusterDeployMachinesTask) OnSyncCompleteFailed(ctx context.Context, cluster *models.SCluster, data jsonutils.JSONObject) {
	t.SetFailed(ctx, cluster, data)
}

func (t *ClusterDeployMachinesTask) SetFailed(ctx context.Context, obj db.IStandaloneModel, reason jsonutils.JSONObject) {
	cluster := obj.(*models.SCluster)
	cluster.SetStatus(t.UserCred, api.ClusterStatusDeployingFail, reason.String())
	t.STask.SetStageFailed(ctx, reason)
	logclient.LogWithStartable(t, obj, logclient.ActionClusterDeploy, reason, t.UserCred, false)
}
