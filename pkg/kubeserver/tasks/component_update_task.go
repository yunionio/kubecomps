package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
)

func init() {
	taskman.RegisterTask(ComponentUpdateTask{})
}

type ComponentUpdateTask struct {
	taskman.STask
}

func (t *ComponentUpdateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	comp := obj.(*models.SComponent)
	cluster, err := comp.GetCluster()
	if err != nil {
		t.onError(ctx, comp, err)
		return
	}
	t.SetStage("OnUpdateComplete", nil)
	taskman.LocalTaskRun(t, func() (jsonutils.JSONObject, error) {
		drv, err := comp.GetDriver()
		if err != nil {
			return nil, err
		}
		settings, err := comp.GetSettings()
		if err != nil {
			return nil, err
		}
		if err := drv.DoUpdate(cluster, settings); err != nil {
			return nil, err
		}
		return nil, nil
	})
}

func (t *ComponentUpdateTask) OnUpdateComplete(ctx context.Context, obj *models.SComponent, data jsonutils.JSONObject) {
	obj.SetStatus(t.UserCred, api.ComponentStatusDeployed, "")
	t.SetStageComplete(ctx, nil)
}

func (t *ComponentUpdateTask) OnUpdateCompleteFailed(ctx context.Context, obj *models.SComponent, reason jsonutils.JSONObject) {
	t.onError(ctx, obj, fmt.Errorf(reason.String()))
}

func (t *ComponentUpdateTask) onError(ctx context.Context, obj *models.SComponent, err error) {
	reason := err.Error()
	obj.SetStatus(t.UserCred, api.ComponentStatusUpdateFail, reason)
	t.STask.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}
