package tasks

import (
	"context"
	"fmt"
	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	"yunion.io/x/kubecomps/pkg/utils/logclient"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
)

func init() {
	taskman.RegisterTask(ClusterCreateTask{})
}

type ClusterCreateTask struct {
	taskman.STask
}

func (t *ClusterCreateTask) getMachines(cluster *models.SCluster) ([]*api.CreateMachineData, error) {
	if !cluster.GetDriver().NeedCreateMachines() {
		return nil, nil
	}
	params := t.GetParams()
	ret := []*api.CreateMachineData{}
	ms := []api.CreateMachineData{}
	if err := params.Unmarshal(&ms, "machines"); err != nil {
		return nil, err
	}
	for _, m := range ms {
		m.ClusterId = cluster.Id
		tmp := m
		ret = append(ret, &tmp)
	}
	return ret, nil
}

func (t *ClusterCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*models.SCluster)
	machines, err := t.getMachines(cluster)
	if err != nil {
		t.onError(ctx, obj, err)
		return
	}
	if len(machines) == 0 {
		t.OnApplyAddonsComplete(ctx, obj, data)
		return
	}
	res := api.ClusterCreateInput{}
	if err := t.GetParams().Unmarshal(&res); err != nil {
		t.onError(ctx, obj, fmt.Errorf("Unmarshal: %v", err))
		return
	}
	// generate certificates if needed
	if err := cluster.GenerateCertificates(ctx, t.GetUserCred()); err != nil {
		t.onError(ctx, obj, fmt.Errorf("GenerateCertificates: %v", err))
		return
	}
	t.CreateMachines(ctx, cluster)
}

func (t *ClusterCreateTask) CreateMachines(ctx context.Context, cluster *models.SCluster) {
	machines, err := t.getMachines(cluster)
	if err != nil {
		t.onError(ctx, cluster, err)
		return
	}
	t.SetStage("OnMachinesCreated", nil)
	cluster.StartCreateMachinesTask(ctx, t.GetUserCred(), machines, t.GetTaskId())
}

func (t *ClusterCreateTask) OnMachinesCreated(ctx context.Context, cluster *models.SCluster, data jsonutils.JSONObject) {
	t.SetStage("OnApplyAddonsComplete", nil)
	cluster.StartApplyAddonsTask(ctx, t.GetUserCred(), t.GetParams(), t.GetTaskId())
}

func (t *ClusterCreateTask) OnMachinesCreatedFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	t.SetFailed(ctx, obj, data)
}

func (t *ClusterCreateTask) OnApplyAddonsComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*models.SCluster)
	t.SetStage("OnSyncStatusComplete", nil)
	cluster.StartSyncStatus(ctx, t.UserCred, t.GetTaskId())
}

func (t *ClusterCreateTask) OnApplyAddonsCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	t.SetFailed(ctx, obj, data)
}

func (t *ClusterCreateTask) OnSyncStatusComplete(ctx context.Context, cluster *models.SCluster, data jsonutils.JSONObject) {
	t.SetStage("OnSyncComplete", nil)
	if err := cluster.StartSyncTask(ctx, t.UserCred, nil, t.GetTaskId()); err != nil {
		t.SetFailed(ctx, cluster, jsonutils.NewString(err.Error()))
	}
}

func (t *ClusterCreateTask) OnSyncStatusCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	t.SetFailed(ctx, obj, data)
}

func (t *ClusterCreateTask) OnSyncComplete(ctx context.Context, cluster *models.SCluster, data jsonutils.JSONObject) {
	logclient.LogWithStartable(t, cluster, logclient.ActionClusterCreate, nil, t.UserCred, true)
	t.SetStageComplete(ctx, nil)
}

func (t *ClusterCreateTask) OnSyncCompleteFailed(ctx context.Context, cluster *models.SCluster, data jsonutils.JSONObject) {
	t.SetFailed(ctx, cluster, data)
}

func (t *ClusterCreateTask) onError(ctx context.Context, cluster db.IStandaloneModel, err error) {
	t.SetFailed(ctx, cluster, jsonutils.NewString(err.Error()))
}

func (t *ClusterCreateTask) SetFailed(ctx context.Context, obj db.IStandaloneModel, reason jsonutils.JSONObject) {
	cluster := obj.(*models.SCluster)
	cluster.SetStatus(t.UserCred, api.ClusterStatusCreateFail, reason.String())
	t.STask.SetStageFailed(ctx, reason)
	logclient.LogWithStartable(t, obj, logclient.ActionClusterCreate, reason, t.UserCred, false)
}
