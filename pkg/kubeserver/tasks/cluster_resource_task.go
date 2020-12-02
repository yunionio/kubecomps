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
	for _, t := range []interface{}{
		ClusterResourceCreateTask{},
		ClusterResourceUpdateTask{},
		ClusterResourceDeleteTask{},
	} {
		taskman.RegisterTask(t)
	}
}

type ClusterResourceBaseTask struct {
	taskman.STask
}

func (t *ClusterResourceBaseTask) getModelManager(obj db.IStandaloneModel) (models.IClusterModel, models.IClusterModelManager) {
	resObj := obj.(models.IClusterModel)
	resMan := resObj.GetModelManager().(models.IClusterModelManager)
	return resObj, resMan
}

type ClusterResourceCreateTask struct {
	ClusterResourceBaseTask
}

func (t *ClusterResourceCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	resObj, resMan := t.getModelManager(obj)
	resObj.SetStatus(t.UserCred, api.ClusterResourceStatusCreating, "create resource")
	t.SetStage("OnCreateComplete", nil)
	taskman.LocalTaskRun(t, func() (jsonutils.JSONObject, error) {
		obj, err := models.CreateRemoteObject(ctx, t.UserCred, resMan, resObj, t.GetParams())
		if err != nil {
			log.Errorf("CreateRemoteObject error: %v", err)
			return nil, errors.Wrap(err, "CreateRemoteObject")
		}
		return jsonutils.Marshal(obj), nil
	})
}

func (t *ClusterResourceCreateTask) OnCreateComplete(ctx context.Context, obj models.IClusterModel, data jsonutils.JSONObject) {
	cAPI := models.GetClusterResAPI()
	t.SetStage("OnResourceSyncComplete", nil)
	if err := cAPI.StartResourceSyncTask(obj, ctx, t.UserCred, data.(*jsonutils.JSONDict), t.GetId()); err != nil {
		t.OnCreateCompleteFailed(ctx, obj, jsonutils.NewString(err.Error()))
	}
}

func (t *ClusterResourceCreateTask) OnCreateCompleteFailed(ctx context.Context, obj models.IClusterModel, reason jsonutils.JSONObject) {
	SetObjectTaskFailed(ctx, t, obj, api.ClusterResourceStatusCreateFail, reason.String())
}

func (t *ClusterResourceCreateTask) OnResourceSyncComplete(ctx context.Context, obj models.IClusterModel, data jsonutils.JSONObject) {
	logclient.LogWithStartable(t, obj, logclient.ActionResourceCreate, nil, t.GetUserCred(), true)
	t.SetStageComplete(ctx, nil)
}

func (t *ClusterResourceCreateTask) OnResourceSyncCompleteFailed(ctx context.Context, obj models.IClusterModel, err jsonutils.JSONObject) {
	logclient.LogWithStartable(t, obj, logclient.ActionResourceCreate, err, t.GetUserCred(), false)
	t.SetStageFailed(ctx, err)
}

type ClusterResourceUpdateTask struct {
	ClusterResourceBaseTask
}

func (t *ClusterResourceUpdateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	resObj, _ := t.getModelManager(obj)
	resObj.SetStatus(t.UserCred, api.ClusterResourceStatusUpdating, "update resource")
	t.SetStage("OnUpdateComplete", nil)
	taskman.LocalTaskRun(t, func() (jsonutils.JSONObject, error) {
		obj, err := models.UpdateRemoteObject(ctx, t.UserCred, resObj, t.GetParams())
		if err != nil {
			log.Errorf("UpdateRemoteObject error: %v", err)
			return nil, errors.Wrap(err, "UpdateRemoteObject")
		}
		return jsonutils.Marshal(obj), nil
	})
}

func (t *ClusterResourceUpdateTask) OnUpdateComplete(ctx context.Context, obj models.IClusterModel, data jsonutils.JSONObject) {
	t.SetStageComplete(ctx, nil)
	logclient.LogWithStartable(t, obj, logclient.ActionResourceUpdate, nil, t.GetUserCred(), true)
}

func (t *ClusterResourceUpdateTask) OnUpdateCompleteFailed(ctx context.Context, obj models.IClusterModel, reason jsonutils.JSONObject) {
	SetObjectTaskFailed(ctx, t, obj, api.ClusterResourceStatusUpdateFail, reason.String())
	logclient.LogWithStartable(t, obj, logclient.ActionResourceUpdate, reason, t.GetUserCred(), false)
}

type ClusterResourceDeleteTask struct {
	ClusterResourceBaseTask
}

func (t *ClusterResourceDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	resObj, resMan := t.getModelManager(obj)
	resObj.SetStatus(t.UserCred, api.ClusterResourceStatusDeleting, "delete resource")
	t.SetStage("OnDeleteComplete", nil)
	taskman.LocalTaskRun(t, func() (jsonutils.JSONObject, error) {
		err := models.DeleteRemoteObject(ctx, t.UserCred, resMan, resObj, t.Params)
		if err != nil {
			log.Errorf("DeleteRemoteObject error: %v", err)
			return nil, errors.Wrap(err, "DeleteRemoteObject")
		}
		return jsonutils.Marshal(obj), nil
	})
}

func (t *ClusterResourceDeleteTask) OnDeleteComplete(ctx context.Context, obj models.IClusterModel, data jsonutils.JSONObject) {
	if err := obj.RealDelete(ctx, t.UserCred); err != nil {
		t.OnDeleteCompleteFailed(ctx, obj, jsonutils.NewString(err.Error()))
		return
	}
	logclient.LogWithStartable(t, obj, logclient.ActionResourceDelete, nil, t.GetUserCred(), true)
	t.SetStageComplete(ctx, nil)
}

func (t *ClusterResourceDeleteTask) OnDeleteCompleteFailed(ctx context.Context, obj models.IClusterModel, reason jsonutils.JSONObject) {
	SetObjectTaskFailed(ctx, t, obj, api.ClusterResourceStatusDeleteFail, reason.String())
	logclient.LogWithStartable(t, obj, logclient.ActionResourceDelete, reason, t.GetUserCred(), false)
}
