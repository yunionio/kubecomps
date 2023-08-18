package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	"yunion.io/x/kubecomps/pkg/kubeserver/models/manager"
	"yunion.io/x/kubecomps/pkg/utils/logclient"
)

type ClusterDeleteMachinesTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ClusterDeleteMachinesTask{})
}

func (t *ClusterDeleteMachinesTask) getDeleteMachines() ([]manager.IMachine, error) {
	machinesData, err := t.GetParams().GetArray("machines")
	if err != nil {
		return nil, err
	}
	machines := []manager.IMachine{}
	for _, obj := range machinesData {
		id, err := obj.GetString()
		if err != nil {
			return nil, err
		}
		machineObj, err := manager.MachineManager().FetchMachineByIdOrName(t.UserCred, id)
		if err != nil {
			return nil, err
		}
		machines = append(machines, machineObj)
	}
	return machines, nil
}

func (t *ClusterDeleteMachinesTask) IsFromClusterDeleteTask(cluster *models.SCluster) bool {
	return cluster.GetStatus() == api.ClusterStatusDeleting
}

func (t *ClusterDeleteMachinesTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*models.SCluster)
	ms, err := t.getDeleteMachines()
	if err != nil {
		t.OnError(ctx, cluster, err.Error())
		return
	}

	mIds := make([]string, 0)
	for _, m := range ms {
		mIds = append(mIds, m.GetId())
	}

	if t.IsFromClusterDeleteTask(cluster) {
		t.OnClusterNodeRemoved(ctx, cluster, data)
		return
	} else {
		t.SetStage("OnClusterNodeRemoved", nil)
		cluster.StartDeployMachinesTask(ctx, t.GetUserCred(), api.ClusterDeployActionRemoveNode, mIds, t.GetTaskId(), false)
	}
}

func (t *ClusterDeleteMachinesTask) OnClusterNodeRemoved(ctx context.Context, cluster *models.SCluster, data jsonutils.JSONObject) {
	ms, err := t.getDeleteMachines()
	if err != nil {
		t.OnError(ctx, cluster, err.Error())
		return
	}
	t.SetStage("OnDeleteMachines", nil)
	if err := cluster.GetDriver().RequestDeleteMachines(ctx, t.UserCred, cluster, ms, t); err != nil {
		t.OnError(ctx, cluster, err.Error())
		return
	}
}

func (t *ClusterDeleteMachinesTask) OnClusterNodeRemovedFailed(ctx context.Context, cluster *models.SCluster, data jsonutils.JSONObject) {
	t.OnError(ctx, cluster, data.String())
}

func (t *ClusterDeleteMachinesTask) OnDeleteMachines(ctx context.Context, cluster *models.SCluster, data jsonutils.JSONObject) {
	logclient.LogWithStartable(t, cluster, logclient.ActionClusterDeleteMachine, nil, t.UserCred, true)
	if t.IsFromClusterDeleteTask(cluster) {
		t.SetStageComplete(ctx, nil)
		return
	}
	t.SetStage("OnSyncStatus", nil)
	cluster.StartSyncStatus(ctx, t.UserCred, t.GetTaskId())
}

func (t *ClusterDeleteMachinesTask) OnDeleteMachinesFailed(ctx context.Context, cluster *models.SCluster, data jsonutils.JSONObject) {
	t.OnError(ctx, cluster, data.String())
}

func (t *ClusterDeleteMachinesTask) OnSyncStatus(ctx context.Context, cluster *models.SCluster, data jsonutils.JSONObject) {
	t.SetStageComplete(ctx, nil)
}

func (t *ClusterDeleteMachinesTask) OnSyncStatusFailed(ctx context.Context, cluster *models.SCluster, data jsonutils.JSONObject) {
	t.OnError(ctx, cluster, data.String())
}

func (t *ClusterDeleteMachinesTask) OnError(ctx context.Context, cluster *models.SCluster, err string) {
	cluster.SetStatus(t.UserCred, api.ClusterStatusError, err)
	t.SetStageFailed(ctx, jsonutils.NewString(err))
	logclient.LogWithStartable(t, cluster, logclient.ActionClusterDeleteMachine, err, t.UserCred, false)
}
