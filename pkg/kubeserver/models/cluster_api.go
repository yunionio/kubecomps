package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
)

var (
	clusterResAPI IClusterResAPI
)

func init() {
	GetClusterResAPI()
}

func GetClusterResAPI() IClusterResAPI {
	if clusterResAPI == nil {
		clusterResAPI = newClusterResAPI()
	}
	return clusterResAPI
}

type IClusterResAPI interface {
	NamespaceScope() INamespaceResAPI

	// StartResourceSyncTask start sync cluster model resource task
	StartResourceSyncTask(obj IClusterModel, ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentId string) error
	// PerformSyncResource sync remote cluster resource to local
	PerformSyncResource(obj IClusterModel, ctx context.Context, userCred mcclient.TokenCredential) error
	// UpdateFromRemoteObject update local db object from remote cluster object
	UpdateFromRemoteObject(obj IClusterModel, ctx context.Context, userCred mcclient.TokenCredential, remoteObj interface{}) error
	// PerformGC cleanup cluster orphan resources
	PerformGC(man IClusterModelManager, ctx context.Context, userCred mcclient.TokenCredential) error
}

type INamespaceResAPI interface {
}

type sClusterResAPI struct {
	namespaceScope INamespaceResAPI
}

func newClusterResAPI() IClusterResAPI {
	a := new(sClusterResAPI)
	a.namespaceScope = newNamespaceResAPI(a)
	return a
}

func (a sClusterResAPI) NamespaceScope() INamespaceResAPI {
	return a.namespaceScope
}

func (a sClusterResAPI) StartResourceSyncTask(obj IClusterModel, ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ClusterResourceSyncTask", obj, userCred, data, parentId, "", nil)
	if err != nil {
		return errors.Wrap(err, "New ClusterResourceSyncTask")
	}
	task.ScheduleRun(nil)
	return nil
}

func (a sClusterResAPI) PerformSyncResource(obj IClusterModel, ctx context.Context, userCred mcclient.TokenCredential) error {
	remoteObj, err := obj.GetRemoteObject()
	if err != nil {
		return errors.Wrap(err, "get remote object")
	}

	if err := a.UpdateFromRemoteObject(obj, ctx, userCred, remoteObj); err != nil {
		return errors.Wrap(err, "update from remote object")
	}
	return nil
}

func (a sClusterResAPI) UpdateFromRemoteObject(obj IClusterModel, ctx context.Context, userCred mcclient.TokenCredential, remoteObj interface{}) error {
	diff, err := db.UpdateWithLock(ctx, obj, func() error {
		if err := obj.UpdateFromRemoteObject(ctx, userCred, remoteObj); err != nil {
			return errors.Wrap(err, "UpdateFromRemoteObject")
		}
		cls, err := obj.GetCluster()
		if err != nil {
			return errors.Wrap(err, "GetCluster")
		}
		man := obj.GetClusterModelManager()
		obj.SetExternalId(man.GetRemoteObjectGlobalId(cls, remoteObj))
		if err := obj.SetStatusByRemoteObject(ctx, userCred, remoteObj); err != nil {
			return errors.Wrap(err, "Set status by remote object")
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "Update from remote object error")
	}
	if _, ok := diff["resource_version"]; ok && len(diff) == 1 {
		// not do OpsLog if only resource_version updated
		return nil
	}
	db.OpsLog.LogSyncUpdate(obj, diff, userCred)
	return nil
}

func (a sClusterResAPI) PerformGC(man IClusterModelManager, ctx context.Context, userCred mcclient.TokenCredential) error {
	subMans := man.(ISyncableManager).GetSubManagers()
	for _, subMan := range subMans {
		if err := a.PerformGC(subMan, ctx, userCred); err != nil {
			return errors.Wrapf(err, "PerformGC for subresource %s", subMan.KeywordPlural())
		}
	}
	log.Infof("Start cluster %s resource GC", man.KeywordPlural())
	objs := make([]interface{}, 0)
	q := man.GetGCQuery()
	if err := db.FetchModelObjects(man, q, &objs); err != nil {
		return errors.Wrapf(err, "FetchModelObjects %s", man.KeywordPlural())
	}
	for _, obj := range objs {
		objPtr := GetObjectPtr(obj).(iPurgeClusterResource)
		if err := objPtr.RealDelete(ctx, userCred); err != nil {
			return errors.Wrapf(err, "delete %s object %s", man.Keyword(), objPtr.GetId())
		} else {
			log.Infof("GC object %s", objPtr.(IClusterModel).LogPrefix())
		}
	}
	log.Infof("End cluster %s resource GC", man.KeywordPlural())
	return nil
}

type sNamespaceResAPI struct {
	clusterResAPI IClusterResAPI
}

func newNamespaceResAPI(a IClusterResAPI) INamespaceResAPI {
	na := new(sNamespaceResAPI)
	na.clusterResAPI = a
	return na
}
