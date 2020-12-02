package tasks

import (
	"context"

	"yunion.io/x/kubecomps/pkg/kubeserver/models"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/models/manager"
)

type ClusterDeployMachinesTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ClusterDeployMachinesTask{})
}

func (t *ClusterDeployMachinesTask) getAddMachines() ([]manager.IMachine, error) {
	msIds := []string{}
	if err := t.Params.Unmarshal(&msIds, models.MachinesDeployIdsKey); err != nil {
		return nil, err
	}
	ms := make([]manager.IMachine, 0)
	for _, id := range msIds {
		m, err := models.MachineManager.FetchMachineById(id)
		if err != nil {
			return nil, err
		}
		ms = append(ms, m)
	}
	return ms, nil
}

func (t *ClusterDeployMachinesTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*models.SCluster)
	ms, err := t.getAddMachines()
	if err != nil {
		t.OnError(ctx, cluster, err.Error())
		return
	}
	t.SetStage("OnDeployMachines", nil)
	if err := cluster.GetDriver().RequestDeployMachines(ctx, t.UserCred, cluster, ms, t); err != nil {
		t.OnError(ctx, cluster, err.Error())
		return
	}

}

func (t *ClusterDeployMachinesTask) OnDeployMachines(ctx context.Context, cluster *models.SCluster, data jsonutils.JSONObject) {
	cluster.StartApplyAddonsTask(ctx, t.UserCred, nil, "")
	t.SetStageComplete(ctx, nil)
}

func (t *ClusterDeployMachinesTask) OnDeployMachinesFailed(ctx context.Context, cluster *models.SCluster, data jsonutils.JSONObject) {
	t.OnError(ctx, cluster, data.String())
}

func (t *ClusterDeployMachinesTask) OnError(ctx context.Context, cluster *models.SCluster, err string) {
	cluster.SetStatus(t.UserCred, api.ClusterStatusError, err)
	t.SetStageFailed(ctx, jsonutils.NewString(err))
}
