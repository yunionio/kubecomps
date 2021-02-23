package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
)

func init() {
	taskman.RegisterTask(MachineBatchCreateTask{})
}

type MachineBatchCreateTask struct {
	taskman.STask
}

func (t *MachineBatchCreateTask) getCreateData() ([]*api.CreateMachineData, error) {
	params := t.GetParams()
	ret := make([]*api.CreateMachineData, 0)
	if err := params.Unmarshal(&ret, "machines"); err != nil {
		return nil, errors.Wrap(err, "unmarshal create machine data")
	}
	return ret, nil
}

func (t *MachineBatchCreateTask) OnInit(ctx context.Context, objs []db.IStandaloneModel, body jsonutils.JSONObject) {
	datas, err := t.getCreateData()
	if err != nil {
		t.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		return
	}

	t.SetStage("OnMachinesCreated", nil)
	for idx, obj := range objs {
		m := obj.(*models.SMachine)
		// TODO: use struct input
		data := jsonutils.Marshal(datas[idx]).(*jsonutils.JSONDict)
		if err := m.StartMachineCreateTask(ctx, t.GetUserCred(), data, t.GetTaskId()); err != nil {
			t.SetStageFailed(ctx, jsonutils.NewString(errors.Wrapf(err, "create machine %s", obj.GetName()).Error()))
			return
		}
	}
}

func (t *MachineBatchCreateTask) OnMachinesCreated(ctx context.Context, objs []db.IStandaloneModel, data *jsonutils.JSONDict) {
	t.SetStageComplete(ctx, nil)
}

func (t *MachineBatchCreateTask) OnMachinesCreatedFailed(ctx context.Context, objs []db.IStandaloneModel, data *jsonutils.JSONDict) {
	t.SetStageFailed(ctx, data)
}
