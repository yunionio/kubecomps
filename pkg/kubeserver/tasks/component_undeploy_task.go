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
	taskman.RegisterTask(ComponentUndeployTask{})
}

type ComponentUndeployTask struct {
	taskman.STask
}

func (t *ComponentUndeployTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	comp := obj.(*models.SComponent)
	cluster, err := comp.GetCluster()
	if err != nil {
		t.onError(ctx, comp, err)
		return
	}
	drv, err := comp.GetDriver()
	if err != nil {
		t.onError(ctx, comp, err)
		return
	}
	settings, err := comp.GetSettings()
	if err != nil {
		t.onError(ctx, comp, err)
		return
	}
	if err := drv.DoDisable(cluster, settings); err != nil {
		t.onError(ctx, comp, err)
		return
	}
	comp.SetStatus(ctx, t.UserCred, api.ComponentStatusInit, "")
	comp.SetEnabled(false)
	t.SetStageComplete(ctx, nil)
}

func (t *ComponentUndeployTask) onError(ctx context.Context, obj *models.SComponent, err error) {
	reason := err.Error()
	obj.SetStatus(ctx, t.UserCred, api.ComponentStatusUndeployFail, reason)
	t.STask.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}
