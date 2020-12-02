package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/models"
)

func init() {
	taskman.RegisterTask(MachineBatchDeleteTask{})
}

type MachineBatchDeleteTask struct {
	taskman.STask
}

func (self *MachineBatchDeleteTask) OnInit(ctx context.Context, objs []db.IStandaloneModel, body jsonutils.JSONObject) {
	self.SetStage("OnDeleteMachines", nil)
	for _, obj := range objs {
		if err := self.doDelete(ctx, obj.(*models.SMachine)); err != nil {
			self.SetStageFailed(ctx, jsonutils.NewString(errors.Wrapf(err, "delete machine %s", obj.GetName()).Error()))
			return
		}
	}
}

func (self *MachineBatchDeleteTask) doDelete(ctx context.Context, machine *models.SMachine) error {
	return machine.StartTerminateTask(ctx, self.GetUserCred(), nil, self.GetTaskId())
}

func (self *MachineBatchDeleteTask) OnDeleteMachines(ctx context.Context, objs []db.IStandaloneModel, data *jsonutils.JSONDict) {
	for _, obj := range objs {
		if err := obj.(*models.SMachine).RealDelete(ctx, self.GetUserCred()); err != nil {
			self.SetStageFailed(ctx, jsonutils.NewString(fmt.Sprintf("Delete machine %s error: %v", obj.GetName(), err)))
			return
		}
	}
	self.SetStageComplete(ctx, nil)
}

func (self *MachineBatchDeleteTask) OnDeleteMachinesFailed(ctx context.Context, objs []db.IStandaloneModel, data *jsonutils.JSONDict) {
	self.SetStageFailed(ctx, data)
}
