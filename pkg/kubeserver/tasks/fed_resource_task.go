package tasks

import (
	"context"
	"database/sql"
	"time"

	corev1 "k8s.io/api/core/v1"

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
		FedResourceUpdateTask{},
		FedResourceSyncTask{},
		FedResourceAttachClusterTask{},
	} {
		taskman.RegisterTask(t)
	}
}

type FedResourceBaseTask struct {
	taskman.STask
}

type FedResourceUpdateTask struct {
	FedResourceBaseTask
}

func (t *FedResourceUpdateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	t.SetStage("OnUpdateComplete", nil)
	fedApi := models.GetFedResAPI()
	fedObj := obj.(models.IFedModel)
	if err := fedApi.StartSyncTask(fedObj, ctx, t.GetUserCred(), t.GetParams(), t.GetTaskId()); err != nil {
		t.OnUpdateCompleteFailed(ctx, fedObj, jsonutils.NewString(err.Error()))
	}
}

func (t *FedResourceUpdateTask) OnUpdateComplete(ctx context.Context, obj models.IFedModel, data jsonutils.JSONObject) {
	t.SetStageComplete(ctx, nil)
	logclient.LogWithStartable(t, obj, logclient.ActionResourceUpdate, nil, t.GetUserCred(), true)
}

func (t *FedResourceUpdateTask) OnUpdateCompleteFailed(ctx context.Context, obj models.IFedModel, reason jsonutils.JSONObject) {
	SetObjectTaskFailed(ctx, t, obj, api.FederatedResourceStatusUpdateFail, reason.String())
	logclient.LogWithStartable(t, obj, logclient.ActionResourceUpdate, reason, t.GetUserCred(), false)
}

type FedResourceSyncTask struct {
	FedResourceBaseTask
}

func (t *FedResourceSyncTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	t.SetStage("OnSyncComplete", nil)
	fedApi := models.GetFedResAPI()
	fedObj := obj.(models.IFedModel)
	fedObj.SetStatus(ctx, t.GetUserCred(), api.FederatedResourceStatusSyncing, "start syncing")
	taskman.LocalTaskRun(t, func() (jsonutils.JSONObject, error) {
		clusters, err := fedApi.GetAttachedClusters(fedObj)
		if err != nil {
			return nil, errors.Wrap(err, "get attached clusters")
		}
		for _, cluster := range clusters {
			input := api.FederatedResourceJointClusterInput{
				ClusterId: cluster.GetId(),
			}
			if err := fedApi.PerformSyncCluster(fedObj, ctx, t.UserCred, input.JSON(input)); err != nil {
				log.Errorf("%s sync to cluster %s(%s) error: %v", fedObj.LogPrefix(), cluster.GetName(), cluster.GetId(), err)
			}
		}
		return nil, nil
	})
}

func (t *FedResourceSyncTask) OnSyncComplete(ctx context.Context, obj models.IFedModel, data jsonutils.JSONObject) {
	obj.SetStatus(ctx, t.GetUserCred(), api.FederatedResourceStatusActive, "sync complete")
	t.SetStageComplete(ctx, nil)
	logclient.LogWithStartable(t, obj, logclient.ActionResourceSync, nil, t.GetUserCred(), true)
}

func (t *FedResourceSyncTask) OnSyncCompleteFailed(ctx context.Context, obj models.IFedModel, reason jsonutils.JSONObject) {
	SetObjectTaskFailed(ctx, t, obj, api.FedreatedResourceStatusSyncFail, reason.String())
	logclient.LogWithStartable(t, obj, logclient.ActionResourceSync, reason, t.GetUserCred(), false)
}

type FedResourceAttachClusterTask struct {
	FedResourceBaseTask
}

func (t *FedResourceAttachClusterTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	fedApi := models.GetFedResAPI()
	fedObj := obj.(models.IFedModel)
	fedNsObj, isNamespace := fedObj.(models.IFedNamespaceModel)
	t.SetStage("OnAttachComplete", nil)
	taskman.LocalTaskRun(t, func() (jsonutils.JSONObject, error) {
		clusterId, err := t.GetParams().GetString("cluster_id")
		if err != nil {
			return nil, errors.Wrapf(err, "get cluster_id from task params %s", t.GetParams())
		}
		if !isNamespace {
			if _, err := fedApi.PerformAttachCluster(fedObj, ctx, t.GetUserCred(), t.GetParams()); err != nil {
				return nil, errors.Wrap(err, "PerformAttachCluster to federated cluster resource")
			}
		}
		if err := t.ensureFedNamespaceAttached(ctx, fedNsObj); err != nil {
			return nil, errors.Wrap(err, "ensureFedNamespaceAttached")
		}
		fedNs, err := fedNsObj.GetFedNamespace()
		if err != nil {
			return nil, errors.Wrap(err, "get resource related federated namespace")
		}
		namespaceIsActive := false
		var nsObj models.IClusterModel
		for i := 0; i < 5; i++ {
			var err error
			nsObj, err = models.GetNamespaceManager().GetByIdOrName(t.UserCred, clusterId, fedNs.GetName())
			if err != nil {
				return nil, errors.Wrapf(err, "get cluster %s namespace %s", clusterId, fedNs.GetName())
			}
			if nsObj.GetStatus() == string(corev1.NamespaceActive) {
				namespaceIsActive = true
				break
			}
			time.Sleep(time.Second * 1)
			log.Warningf("%s status %s != %s", nsObj.LogPrefix(), nsObj.GetStatus(), corev1.NamespaceActive)
		}
		if !namespaceIsActive {
			return nil, errors.Errorf("Wait cluster %s namespace active timeout", clusterId)
		}
		data := t.GetParams()
		data.Set("namespace_id", jsonutils.NewString(nsObj.GetId()))
		data.Set("namespace_name", jsonutils.NewString(nsObj.GetName()))
		if _, err := fedApi.PerformAttachCluster(fedNsObj, ctx, t.GetUserCred(), data); err != nil {
			return nil, errors.Wrap(err, "PerformAttachCluster to federated namespace resource")
		}
		return nil, nil
	})
}

func (t *FedResourceAttachClusterTask) ensureFedNamespaceAttached(ctx context.Context, obj models.IFedNamespaceModel) error {
	// attach federated naemspace to related cluster is not exists
	fedNs, err := obj.GetFedNamespace()
	if err != nil {
		return errors.Wrap(err, "get federated namespace")
	}
	input := new(api.FederatedNamespaceAttachClusterInput)
	if err := t.GetParams().Unmarshal(input); err != nil {
		return errors.Wrap(err, "unmarshal params to federated namespace input")
	}
	clusterId := input.ClusterId
	if clusterId == "" {
		return errors.Wrapf(err, "get cluster_id from task params %s", t.GetParams())
	}
	isAttached, err := models.GetFedResAPI().IsAttach2Cluster(fedNs, clusterId)
	if err != nil {
		return errors.Wrapf(err, "check cluster %s is attached to federated namespace %s", clusterId, fedNs.LogPrefix())
	}
	if isAttached {
		nsObj, err := models.GetNamespaceManager().GetByIdOrName(t.UserCred, clusterId, fedNs.GetName())
		if err != nil {
			if errors.Cause(err) != sql.ErrNoRows {
				return errors.Wrapf(err, "get cluster %s namespace %s", clusterId, fedNs.GetName())
			}
		}
		if nsObj == nil {
			// cluster namespace attached in db records, but not exists current now, sync it
			if _, err := fedNs.PerformSyncCluster(ctx, t.GetUserCred(), jsonutils.NewDict(), &input.FederatedResourceJointClusterInput); err != nil {
				return errors.Wrap(err, "perform sync cluster")
			}
		}
	} else {
		if _, err := fedNs.PerformAttachCluster(ctx, t.GetUserCred(), jsonutils.NewDict(), input); err != nil {
			return errors.Wrap(err, "perform attach federated namespace to cluster")
		}
	}
	return nil
}

func (t *FedResourceAttachClusterTask) OnAttachComplete(ctx context.Context, obj models.IFedModel, data jsonutils.JSONObject) {
	t.SetStageComplete(ctx, nil)
}

func (t *FedResourceAttachClusterTask) OnAttachCompleteFailed(ctx context.Context, obj models.IFedModel, reason jsonutils.JSONObject) {
	SetObjectTaskFailed(ctx, t, obj, api.FederatedResourceStatusUpdateFail, reason.String())
	logclient.LogWithStartable(t, obj, logclient.ActionResourceAttach, reason, t.GetUserCred(), false)
}
