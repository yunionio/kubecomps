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

type ClusterCreateMachinesTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ClusterCreateMachinesTask{})
}

func (t *ClusterCreateMachinesTask) getMachines(cluster *models.SCluster) ([]*api.CreateMachineData, error) {
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

func (t *ClusterCreateMachinesTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	cluster := obj.(*models.SCluster)
	machines, err := t.getMachines(cluster)
	if err != nil {
		t.onError(ctx, cluster, err)
		return
	}
	t.SetStage("OnMachinesCreated", nil)
	if err := t.createMachines(ctx, cluster, machines); err != nil {
		t.onError(ctx, cluster, err)
		return
	}
}

func (t *ClusterCreateMachinesTask) createMachines(ctx context.Context, cluster *models.SCluster, ms []*api.CreateMachineData) error {
	return cluster.CreateMachines(ctx, t.GetUserCred(), ms, t)
}

func (t *ClusterCreateMachinesTask) OnMachinesCreated(ctx context.Context, cluster *models.SCluster, data jsonutils.JSONObject) {
	logclient.LogWithStartable(t, cluster, logclient.ActionClusterCreateMachines, nil, t.UserCred, true)
	t.SetStageComplete(ctx, nil)
}

func (t *ClusterCreateMachinesTask) OnMachinesCreatedFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	t.onError(ctx, obj.(*models.SCluster), fmt.Errorf(data.String()))
}

func (t *ClusterCreateMachinesTask) onError(ctx context.Context, cluster *models.SCluster, err error) {
	SetObjectTaskFailed(ctx, t, cluster, api.ClusterStatusCreateMachineFail, err.Error())
	logclient.LogWithStartable(t, cluster, logclient.ActionClusterCreateMachines, err.Error(), t.UserCred, false)
}
