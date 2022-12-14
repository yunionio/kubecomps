package tasks

import (
	"context"
	"fmt"
	"yunion.io/x/jsonutils"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
)

func init() {
	taskman.RegisterTask(ComponentDeployTask{})
}

type ComponentDeployTask struct {
	taskman.STask
}

func (t *ComponentDeployTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	comp := obj.(*models.SComponent)
	cluster, err := comp.GetCluster()
	if err != nil {
		t.onError(ctx, comp, err)
		return
	}
	t.SetStage("OnDeployComplete", nil)
	taskman.LocalTaskRun(t, func() (jsonutils.JSONObject, error) {
		drv, err := comp.GetDriver()
		if err != nil {
			return nil, err
		}
		settings, err := comp.GetSettings()
		if err != nil {
			return nil, err
		}

		if settings.Namespace == models.MonitorNamespace {
			// create oidc secret
			secret, err := models.MonitorComponentManager.CreateOIDCSecret(cluster, "", "")
			if err != nil {
				return nil, err
			}
			if settings.Monitor.Grafana.OAuth == nil {
				settings.Monitor.Grafana.OAuth = &api.ComponentSettingMonitorGrafanaOAuth{}
			}
			settings.Monitor.Grafana.OAuth.ClientId = secret.ClientId
			settings.Monitor.Grafana.OAuth.ClientSecret = secret.SAccessKeySecretBlob.Secret
		}

		if err := drv.DoEnable(cluster, settings); err != nil {
			return nil, err
		}
		return nil, nil
	})
}

func (t *ComponentDeployTask) OnDeployComplete(ctx context.Context, obj *models.SComponent, data jsonutils.JSONObject) {
	obj.SetStatus(t.UserCred, api.ComponentStatusDeployed, "")
	t.SetStageComplete(ctx, nil)
}

func (t *ComponentDeployTask) OnDeployCompleteFailed(ctx context.Context, obj *models.SComponent, reason jsonutils.JSONObject) {
	t.onError(ctx, obj, fmt.Errorf(reason.String()))
}

func (t *ComponentDeployTask) onError(ctx context.Context, obj *models.SComponent, err error) {
	reason := err.Error()
	obj.SetStatus(t.UserCred, api.ComponentStatusDeployFail, reason)
	t.STask.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}
