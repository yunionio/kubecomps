package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
)

func init() {
	taskman.RegisterTask(MachineTerminateTask{})
}

type MachineTerminateTask struct {
	taskman.STask
}

func (t *MachineTerminateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	machine := obj.(*models.SMachine)

	driver := machine.GetDriver()
	session, err := models.MachineManager.GetSession()
	if err != nil {
		t.OnError(ctx, machine, err)
		return
	}
	err = driver.TerminateResource(session, machine)
	if err != nil {
		t.OnError(ctx, machine, err)
		return
	}
	t.SetStageComplete(ctx, nil)
}

func (t *MachineTerminateTask) OnError(ctx context.Context, machine *models.SMachine, err error) {
	machine.SetStatus(ctx, t.UserCred, api.MachineStatusTerminateFail, err.Error())
	t.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}
