package tasks

import (
	"context"
	"fmt"

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
	t.SetStage("OnPrepared", nil)
	taskman.LocalTaskRun(t, func() (jsonutils.JSONObject, error) {
		return nil, t.doPrepare(ctx, obj, data)
	})
}

func (t *MachinePrepareTask) doPrepare(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) error {
	machine := obj.(*models.SMachine)
	param := t.GetParams()

	prepareData := new(api.MachinePrepareInput)
	if err := param.Unmarshal(prepareData); err != nil {
		return err
	}

	cluster, err := machine.GetCluster()
	if err != nil {
		return err
	}
	prepareData, err = cluster.FillMachinePrepareInput(prepareData)
	if err != nil {
		return errors.Wrap(err, "fill prepare input data")
	}

	prepareData.InstanceId = machine.ResourceId
	driver := machine.GetDriver()
	session, err := models.MachineManager.GetSession()
	if err != nil {
		return errors.Wrap(err, "get client session")
	}

	log.Infof("Start prepare resource, data: %s", jsonutils.Marshal(prepareData))
	if _, err := driver.PrepareResource(session, machine, prepareData); err != nil {
		return errors.Wrap(err, "prepare resource")
	}

	ip, err := driver.GetPrivateIP(session, machine.GetResourceId())
	if err != nil {
		return errors.Wrapf(err, "Get resource %s private ip", machine.GetResourceId())
	}

	if err := machine.SetPrivateIP(ip); err != nil {
		return errors.Wrapf(err, "Set machine private ip %s", ip)
	}
	machine.SetStatus(ctx, t.UserCred, api.MachineStatusRunning, "")
	return nil
}

func (t *MachinePrepareTask) OnPrepared(ctx context.Context, machine db.IStandaloneModel, data jsonutils.JSONObject) {
	t.SetStageComplete(ctx, nil)
	logclient.LogWithStartable(t, machine, logclient.ActionMachinePrepare, nil, t.UserCred, true)
}

func (t *MachinePrepareTask) OnPreparedFailed(ctx context.Context, machine *models.SMachine, data jsonutils.JSONObject) {
	t.OnError(ctx, machine, fmt.Errorf(data.String()))
}

func (t *MachinePrepareTask) OnError(ctx context.Context, machine *models.SMachine, err error) {
	machine.SetStatus(ctx, t.UserCred, api.MachineStatusPrepareFail, err.Error())
	t.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
	logclient.LogWithStartable(t, machine, logclient.ActionMachinePrepare, err.Error(), t.UserCred, false)
}
