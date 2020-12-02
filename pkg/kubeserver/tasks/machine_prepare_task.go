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
	taskman.RegisterTask(MachinePrepareTask{})
}

type MachinePrepareTask struct {
	taskman.STask
}

func (t *MachinePrepareTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	machine := obj.(*models.SMachine)
	param := t.GetParams()

	prepareData := new(api.MachinePrepareInput)
	if err := param.Unmarshal(prepareData); err != nil {
		t.OnError(ctx, machine, err)
		return
	}

	cluster, err := machine.GetCluster()
	if err != nil {
		t.OnError(ctx, machine, err)
		return
	}
	prepareData, err = cluster.FillMachinePrepareInput(prepareData)
	if err != nil {
		t.OnError(ctx, machine, err)
		return
	}

	prepareData.InstanceId = machine.ResourceId
	driver := machine.GetDriver()
	session, err := models.MachineManager.GetSession()
	if err != nil {
		t.OnError(ctx, machine, err)
		return
	}
	log.Infof("Start PrepareResource: %#v", prepareData)
	_, err = driver.PrepareResource(session, machine, prepareData)
	if err != nil {
		t.OnError(ctx, machine, err)
		return
	}
	ip, err := driver.GetPrivateIP(session, machine.GetResourceId())
	if err != nil {
		t.OnError(ctx, machine, errors.Wrapf(err, "Get resource %s private ip", machine.GetResourceId))
		return
	}
	if err := machine.SetPrivateIP(ip); err != nil {
		t.OnError(ctx, machine, errors.Wrapf(err, "Set machine private ip %s", ip))
		return
	}
	machine.SetStatus(t.UserCred, api.MachineStatusRunning, "")

	t.SetStageComplete(ctx, nil)
	logclient.LogWithStartable(t, machine, logclient.ActionMachinePrepare, nil, t.UserCred, true)
}

func (t *MachinePrepareTask) OnError(ctx context.Context, machine *models.SMachine, err error) {
	machine.SetStatus(t.UserCred, api.MachineStatusPrepareFail, err.Error())
	t.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
	logclient.LogWithStartable(t, machine, logclient.ActionMachinePrepare, err.Error(), t.UserCred, false)
}
