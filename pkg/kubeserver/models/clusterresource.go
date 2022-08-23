package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/version"
	"strconv"
	"strings"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubectl/pkg/scheme"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
	"yunion.io/x/kubecomps/pkg/kubeserver/k8s/common/model"
	"yunion.io/x/kubecomps/pkg/kubeserver/models/manager"
)

// +onecloud:swagger-gen-ignore
type SClusterResourceBaseManager struct {
	db.SStatusDomainLevelResourceBaseManager
	db.SExternalizedResourceBaseManager

	SK8sOwnedResourceBaseManager

	*SSyncableManager
	// resourceName is kubernetes resource name
	resourceName string
	// groupName is kubernetes resource group
	groupName string
	// versionName string kubernetes resource version
	versionName string
	// kindName is kubernetes resource kind
	kindName string
	// RawObject is kubernetes runtime object
	rawObject runtime.Object
}

func NewClusterResourceBaseManager(
	dt interface{},
	tableName string,
	keyword string,
	keywordPlural string,
	resName string,
	groupName string,
	versionName string,
	kind string,
	object runtime.Object) SClusterResourceBaseManager {
	return SClusterResourceBaseManager{
		SStatusDomainLevelResourceBaseManager: db.NewStatusDomainLevelResourceBaseManager(
			dt, tableName, keyword, keywordPlural),
		resourceName:     resName,
		groupName:        groupName,
		versionName:      versionName,
		kindName:         kind,
		rawObject:        object,
		SSyncableManager: newSyncableManager(),
	}
}

type SClusterResourceBase struct {
	db.SStatusDomainLevelResourceBase
	db.SExternalizedResourceBase

	ClusterId string `width:"36" charset:"ascii" nullable:"false" index:"true" list:"user"`
	// ResourceVersion is k8s remote object resourceVersion
	ResourceVersion string `width:"36" charset:"ascii" nullable:"false" list:"user"`
}

type IClusterModelManager interface {
	db.IDomainLevelModelManager

	IsNamespaceScope() bool
	GetK8sResourceInfo(version *version.Info) model.K8sResourceInfo
	IsRemoteObjectLocalExist(userCred mcclient.TokenCredential, cluster *SCluster, obj interface{}) (IClusterModel, bool, error)
	NewFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, obj interface{}) (IClusterModel, error)
	ListRemoteObjects(cli *client.ClusterManager) ([]interface{}, error)
	GetRemoteObjectGlobalId(cluster *SCluster, obj interface{}) string

	NewRemoteObjectForCreate(model IClusterModel, cli *client.ClusterManager, data jsonutils.JSONObject) (interface{}, error)
	CreateRemoteObject(model IClusterModel, cli *client.ClusterManager, remoteObj interface{}) (interface{}, error)

	InitOwnedManager(man IClusterModelManager)

	GetGCQuery() *sqlchemy.SQuery

	FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []interface{}, fields stringutils2.SSortedStrings, isList bool) []interface{}
}

type IClusterModel interface {
	db.IStatusDomainLevelModel

	// LogPrefix return db object log short prefix string
	LogPrefix() string
	GetExternalId() string
	SetExternalId(idStr string)

	GetClusterModelManager() IClusterModelManager
	SetName(name string)
	GetClusterId() string
	GetCluster() (*SCluster, error)
	SetCluster(userCred mcclient.TokenCredential, cluster *SCluster)
	SetStatus(userCred mcclient.TokenCredential, status string, reason string) error
	NewRemoteObjectForUpdate(cli *client.ClusterManager, remoteObj interface{}, data jsonutils.JSONObject) (interface{}, error)
	SetStatusByRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, remoteObj interface{}) error
	UpdateFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, extObj interface{}) error
	GetClusterClient() (*client.ClusterManager, error)
	RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error
	GetDetails(cli *client.ClusterManager, baseDetails interface{}, k8sObj runtime.Object, isList bool) interface{}

	// RemoteObject operator interfaces
	// GetRemoteObject get remote object from cluster
	GetRemoteObject() (interface{}, error)
	// UpdateRemoteObject update remote object inside cluster
	UpdateRemoteObject(remoteObj interface{}) (interface{}, error)
	// DeleteRemoteObject delete remote object inside cluster
	DeleteRemoteObject() error
}

func (m *SClusterResourceBaseManager) InitOwnedManager(man IClusterModelManager) {
	m.SK8sOwnedResourceBaseManager = newK8sOwnedResourceManager(man)
}

func (_ SClusterResourceBaseManager) GetOwnerModel(userCred mcclient.TokenCredential, manager model.IModelManager, cluster model.ICluster, namespace string, name string) (model.IOwnerModel, error) {
	if namespace != "" {
		nsObj, err := GetNamespaceManager().GetByIdOrName(userCred, cluster.GetId(), namespace)
		if err != nil {
			return nil, errors.Wrapf(err, "get namespace %s", namespace)
		}
		namespace = nsObj.GetId()
	}
	modelObj, err := FetchClusterResourceByIdOrName(manager.(IClusterModelManager), userCred, cluster.GetId(), namespace, name)
	if err != nil {
		return nil, err
	}
	return modelObj.(model.IOwnerModel), nil
}

func (m *SClusterResourceBaseManager) GetGCQuery() *sqlchemy.SQuery {
	clusterIds := GetClusterManager().Query("id").SubQuery()
	q := m.Query().NotIn("cluster_id", clusterIds)
	return q
}

func (m SClusterResourceBaseManager) IsNamespaceScope() bool {
	return false
}

func (m SClusterResourceBaseManager) GetK8sResourceInfo(version *version.Info) model.K8sResourceInfo {
	// FIXME: set Resource Base Manager info correctly
	if version != nil {
		if m.kindName == api.KindNameIngress {
			s := SIngressManager{SNamespaceResourceBaseManager{m}}
			return s.GetK8sResourceInfo(version)
		}
		if m.kindName == api.KindNameReplicaSet {
			s := SReplicaSetManager{SNamespaceResourceBaseManager{m}}
			return s.GetK8sResourceInfo(version)
		}
	}

	return model.K8sResourceInfo{
		ResourceName: m.resourceName,
		Group:        m.groupName,
		Version:      m.versionName,
		KindName:     m.kindName,
		Object:       m.rawObject,
	}
}

func (m *SClusterResourceBaseManager) FilterBySystemAttributes(q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	q = m.SStatusDomainLevelResourceBaseManager.FilterBySystemAttributes(q, userCred, query, scope)
	input := new(api.ClusterResourceListInput)
	if query != nil {
		query.Unmarshal(input)
	}
	isSystem := false
	if input.System != nil && *input.System {
		var isAllow bool
		isSystem = *input.System
		if consts.IsRbacEnabled() {
			allowScope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), m.KeywordPlural(), policy.PolicyActionList, "system")
			if !scope.HigherThan(allowScope) {
				isAllow = true
			}
		} else {
			if userCred.HasSystemAdminPrivilege() {
				isAllow = true
			}
		}
		if !isAllow {
			isSystem = false
		}
	}
	if !isSystem {
		if sysCls, _ := ClusterManager.GetSystemCluster(); sysCls != nil {
			// make system cluster resource can be getted
			if input.Cluster != sysCls.GetName() && input.Cluster != sysCls.GetId() {
				q = q.NotEquals("cluster_id", sysCls.GetId())
			}
		}
	}
	return q
}

func (m SClusterResourceBaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.ClusterResourceCreateInput) (*api.ClusterResourceCreateInput, error) {
	vData, err := m.SStatusDomainLevelResourceBaseManager.ValidateCreateData(ctx, userCred, ownerCred, query, data.StatusDomainLevelResourceCreateInput)
	if err != nil {
		return nil, err
	}
	data.StatusDomainLevelResourceCreateInput = vData

	if data.ClusterId == "" {
		return nil, httperrors.NewNotEmptyError("cluster is empty")
	}
	clsObj, err := ClusterManager.FetchByIdOrName(userCred, data.ClusterId)
	if err != nil {
		return nil, NewCheckIdOrNameError("cluster", data.ClusterId, err)
	}
	data.ClusterId = clsObj.GetId()

	return data, nil
}

func FetchClusterResourceByName(manager IClusterModelManager, userCred mcclient.IIdentityProvider, clusterId string, namespaceId string, resId string) (IClusterModel, error) {
	if len(clusterId) == 0 {
		return nil, errors.Errorf("cluster id must provided")
	}
	q := manager.Query()
	q = manager.FilterByName(q, resId)
	q = q.Equals("cluster_id", clusterId)
	if manager.IsNamespaceScope() && namespaceId != "" {
		q = q.Equals("namespace_id", namespaceId)
	}
	count, err := q.CountWithError()
	if err != nil {
		return nil, errors.Wrap(err, "first count with error")
	}
	// if count > 0 && userCred != nil {
	// 	q = manager.FilterByOwner(q, userCred, manager.NamespaceScope())
	// 	//q = manager.FilterBySystemAttributes(q, nil, nil, manager.ResourceScope())
	// 	count, err = q.CountWithError()
	// 	if err != nil {
	// 		return nil, errors.Wrap(err, "filter by owner")
	// 	}
	// }
	if count == 1 {
		obj, err := db.NewModelObject(manager)
		if err != nil {
			return nil, errors.Wrap(err, "NewModelObject")
		}
		if err := q.First(obj); err != nil {
			return nil, err
		} else {
			return obj.(IClusterModel), nil
		}
	} else if count > 1 {
		return nil, sqlchemy.ErrDuplicateEntry
	} else {
		return nil, sql.ErrNoRows
	}
}

func FetchClusterResourceById(manager IClusterModelManager, clusterId string, namespaceId string, resId string) (IClusterModel, error) {
	if len(clusterId) == 0 {
		return nil, errors.Errorf("cluster id must provided")
	}
	/*
	 * if manager.IsNamespaceScope() && namespaceId == "" {
	 *     return nil, errors.Errorf("namespace id must provided for %s", manager.Keyword())
	 * }
	 */
	q := manager.Query()
	q = manager.FilterById(q, resId)
	q = q.Equals("cluster_id", clusterId)
	if manager.IsNamespaceScope() && namespaceId != "" {
		q = q.Equals("namespace_id", namespaceId)
	}
	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}
	if count == 1 {
		obj, err := db.NewModelObject(manager)
		if err != nil {
			return nil, err
		}
		if err := q.First(obj); err != nil {
			return nil, err
		} else {
			return obj.(IClusterModel), nil
		}
	} else if count > 1 {
		return nil, sqlchemy.ErrDuplicateEntry
	} else {
		return nil, sql.ErrNoRows
	}
}

func FetchClusterResourceByIdOrName(manager IClusterModelManager, userCred mcclient.IIdentityProvider, clusterId string, namespaceId string, resId string) (IClusterModel, error) {
	if stringutils2.IsUtf8(resId) {
		return FetchClusterResourceByName(manager, userCred, clusterId, namespaceId, resId)
	}
	obj, err := FetchClusterResourceById(manager, clusterId, namespaceId, resId)
	if err == sql.ErrNoRows {
		return FetchClusterResourceByName(manager, userCred, clusterId, namespaceId, resId)
	} else {
		return obj, err
	}
}

func (m *SClusterResourceBaseManager) GetByIdOrName(userCred mcclient.IIdentityProvider, clusterId string, resId string) (IClusterModel, error) {
	return FetchClusterResourceByIdOrName(m, userCred, clusterId, "", resId)
}

func (m *SClusterResourceBaseManager) GetByName(userCred mcclient.IIdentityProvider, clusterId string, resId string) (IClusterModel, error) {
	return FetchClusterResourceByName(m, userCred, clusterId, "", resId)
}

func NewObjectMeta(res IClusterModel) (api.ObjectMeta, error) {
	kObj, err := GetK8sObject(res)
	if err != nil {
		return api.ObjectMeta{}, errors.Wrap(err, "get k8s object")
	}
	cluster, err := res.GetCluster()
	if err != nil {
		return api.ObjectMeta{}, errors.Wrap(err, "get cluster")
	}
	return model.NewObjectMeta(kObj, cluster)
}

func (res SClusterResourceBase) GetObjectMeta() (api.ObjectMeta, error) {
	return NewObjectMeta(&res)
}

func (res *SClusterResourceBase) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if err := res.SStatusDomainLevelResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data); err != nil {
		return err
	}
	input := new(api.ClusterResourceCreateInput)
	if err := data.Unmarshal(input); err != nil {
		return errors.Wrap(err, "cluster resource unmarshal data")
	}
	res.ClusterId = input.ClusterId
	return nil
}

func (res *SClusterResourceBase) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	res.SStatusDomainLevelResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	if err := res.StartCreateTask(res, ctx, userCred, ownerId, data.(*jsonutils.JSONDict), ""); err != nil {
		log.Errorf("StartCreateTask error: %v", err)
	}
}

func (res *SClusterResourceBase) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	res.SStatusDomainLevelResourceBase.PostUpdate(ctx, userCred, query, data)
	if err := res.StartUpdateTask(res, ctx, userCred, data.(*jsonutils.JSONDict), ""); err != nil {
		log.Errorf("StartUpdateTask error: %v", err)
	}
}

func (res *SClusterResourceBase) PostDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	res.SStatusDomainLevelResourceBase.PostDelete(ctx, userCred)
	if err := res.StartDeleteTask(res, ctx, userCred, jsonutils.NewDict(), ""); err != nil {
		log.Errorf("StartDeleteTask error: %v", err)
	}
}

func (r *SClusterResourceBase) GetUniqValues() jsonutils.JSONObject {
	return jsonutils.Marshal(map[string]string{
		"cluster_id": r.ClusterId,
	})
}

func (m *SClusterResourceBaseManager) FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject {
	clusterId, err := data.GetString("cluster_id")
	if err != nil {
		panic(fmt.Sprintf("get cluster_id from data %s error: %v", data, err))
	}
	return jsonutils.Marshal(map[string]string{
		"cluster_id": clusterId,
	})
}

func (m *SClusterResourceBaseManager) FilterByUniqValues(q *sqlchemy.SQuery, values jsonutils.JSONObject) *sqlchemy.SQuery {
	clusterId, _ := values.GetString("cluster_id")
	if len(clusterId) > 0 {
		q = q.Equals("cluster_id", clusterId)
	}
	return q
}

func (res *SClusterResourceBase) GetCluster() (*SCluster, error) {
	return GetClusterById(res.ClusterId)
}

func GetClusterById(id string) (*SCluster, error) {
	obj, err := ClusterManager.FetchById(id)
	if err != nil {
		return nil, errors.Wrapf(err, "fetch cluster %s", id)
	}
	return obj.(*SCluster), nil
}

func (res *SClusterResourceBase) SetCluster(userCred mcclient.TokenCredential, cls *SCluster) {
	res.ClusterId = cls.GetId()
	res.SyncCloudDomainId(userCred, cls.GetOwnerId())
}

func (res *SClusterResourceBase) GetClusterClient() (*client.ClusterManager, error) {
	cls, err := res.GetCluster()
	if err != nil {
		return nil, err
	}
	return client.GetManagerByCluster(cls)
}

func (res *SClusterResourceBase) AllowPerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, res, "purge")
}

func (res *SClusterResourceBase) PerformPurge(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, res.RealDelete(ctx, userCred)
}

func GetClusterClient(clsId string) (*client.ClusterManager, error) {
	cls, err := GetClusterById(clsId)
	if err != nil {
		return nil, err
	}
	return client.GetManagerByCluster(cls)
}

func GetClusterModelObjects(man IClusterModelManager, cluster *SCluster) ([]IClusterModel, error) {
	q := man.Query().Equals("cluster_id", cluster.GetId())
	rows, err := q.Rows()
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	ret := make([]IClusterModel, 0)
	defer rows.Close()
	for rows.Next() {
		m, err := db.NewModelObject(man)
		if err != nil {
			return nil, errors.Wrapf(err, "NewModelObject of %s", man.Keyword())
		}
		if err := q.Row2Struct(rows, m); err != nil {
			return nil, errors.Wrapf(err, "Row2Struct of %s", man.Keyword())
		}
		ret = append(ret, m.(IClusterModel))
	}
	return ret, nil
}

func SyncClusterResources(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	cluster *SCluster,
	resMans ...ISyncableManager,
) error {
	if len(resMans) == 0 {
		return nil
	}
	for _, man := range resMans {
		// set cluster sync_message
		cluster.SaveSyncMessage(cluster, fmt.Sprintf("syncing_%s", man.KeywordPlural()))
		if ret := syncClusterResources(ctx, man, userCred, cluster); ret.IsError() {
			err := errors.Errorf("Sync cluster %s resource %s error: %v", cluster.GetName(), man.KeywordPlural(), ret.Result())
			cluster.MarkErrorSync(ctx, cluster, err)
			return err
		} else {
			log.Infof("Sync cluster %s resource %s completed: %v", cluster.GetName(), man.KeywordPlural(), ret.Result())
		}
		if err := SyncClusterResources(ctx, userCred, cluster, man.GetSubManagers()...); err != nil {
			return errors.Wrapf(err, "Sync sub resources")
		}
	}
	return nil
}

func syncClusterResources(
	ctx context.Context,
	man IClusterModelManager,
	userCred mcclient.TokenCredential,
	cluster *SCluster) compare.SyncResult {

	localObjs := make([]db.IModel, 0)
	remoteObjs := make([]interface{}, 0)
	syncResult := compare.SyncResult{}

	clsCli, err := client.GetManagerByCluster(cluster)
	if err != nil {
		syncResult.Error(errors.Wrapf(err, "Get cluster %s client", cluster.GetName()))
		return syncResult
	}

	listObjs, err := man.ListRemoteObjects(clsCli)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}
	dbObjs, err := GetClusterModelObjects(man, cluster)
	if err != nil {
		syncResult.Error(errors.Wrapf(err, "get %s db objects", man.Keyword()))
		return syncResult
	}

	for i := range dbObjs {
		dbObj := dbObjs[i]
		if taskman.TaskManager.IsInTask(dbObj) {
			log.Warningf("cluster %s resource %s object %s is in task, exit this sync task", dbObj.GetClusterId(), dbObj.Keyword(), dbObj.GetName())
			syncResult.Error(fmt.Errorf("object %s/%s is in task", dbObj.Keyword(), dbObj.GetName()))
			return syncResult
		}
	}

	removed := make([]IClusterModel, 0)
	commondb := make([]IClusterModel, 0)
	commonext := make([]interface{}, 0)
	added := make([]interface{}, 0)

	getGlobalIdF := func(obj interface{}) string {
		return man.GetRemoteObjectGlobalId(cluster, obj)
	}

	if err := CompareRemoteObjectSets(
		dbObjs, listObjs,
		getGlobalIdF,
		&removed, &commondb, &commonext, &added); err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		if err := SyncRemovedClusterResource(ctx, userCred, removed[i]); err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	for i := 0; i < len(commondb); i += 1 {
		if err := SyncUpdatedClusterResource(ctx, userCred, man, commondb[i], commonext[i]); err != nil {
			syncResult.UpdateError(err)
		} else {
			localObjs = append(localObjs, commondb[i])
			remoteObjs = append(remoteObjs, commonext[i])
			syncResult.Update()
		}
	}

	for i := 0; i < len(added); i += 1 {
		newObj, err := NewFromRemoteObject(ctx, userCred, man, cluster, added[i])
		if err != nil {
			syncResult.AddError(errors.Wrapf(err, "add object"))
		} else {
			localObjs = append(localObjs, newObj)
			remoteObjs = append(remoteObjs, added[i])
			syncResult.Add()
		}
	}
	return syncResult
}

func NewFromRemoteObject(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	man IClusterModelManager,
	cluster *SCluster,
	obj interface{}) (db.IModel, error) {
	lockman.LockClass(ctx, man, db.GetLockClassKey(man, userCred))
	defer lockman.ReleaseClass(ctx, man, db.GetLockClassKey(man, userCred))

	localObj, exist, err := man.IsRemoteObjectLocalExist(userCred, cluster, obj)
	if err != nil {
		return nil, errors.Wrap(err, "check IsRemoteObjectLocalExist")
	}
	if exist {
		return nil, httperrors.NewDuplicateResourceError("%s %v already exists", man.Keyword(), localObj.GetName())
	}
	dbObj, err := man.NewFromRemoteObject(ctx, userCred, cluster, obj)
	if err != nil {
		return nil, errors.Wrapf(err, "NewFromRemoteObject %s", man.Keyword())
	}
	if err := man.TableSpec().InsertOrUpdate(ctx, dbObj); err != nil {
		return nil, errors.Wrapf(err, "Insert %#v to database", dbObj)
	}
	if err := GetClusterResAPI().UpdateFromRemoteObject(dbObj, ctx, userCred, obj); err != nil {
		return nil, errors.Wrap(err, "NewFromRemoteObject.UpdateFromRemoteObject")
	}
	return dbObj, nil
}

func SyncRemovedClusterResource(ctx context.Context, userCred mcclient.TokenCredential, dbObj IClusterModel) error {
	lockman.LockObject(ctx, dbObj)
	defer lockman.ReleaseObject(ctx, dbObj)

	status := dbObj.GetStatus()
	if strings.HasSuffix(status, "_fail") || strings.HasSuffix(status, "_failed") {
		log.Warningf("object %s status is %s, skip deleted it", dbObj.LogPrefix(), dbObj.GetStatus())
		return nil
	}
	if err := dbObj.RealDelete(ctx, userCred); err != nil {
		return errors.Wrapf(err, "SyncRemovedClusterResource ")
	}
	log.Infof("Delete local db record %s", dbObj.LogPrefix())

	/*if err := dbObj.ValidateDeleteCondition(ctx); err != nil {
		err := errors.Wrapf(err, "ValidateDeleteCondition")
		dbObj.SetStatus(userCred, api.ClusterResourceStatusDeleteFail, err.Error())
		return err
	}

	if err := db.CustomizeDelete(dbObj, ctx, userCred, nil, nil); err != nil {
		err := errors.Wrap(err, "CustomizeDelete")
		dbObj.SetStatus(userCred, api.ClusterStatusDeleteFail, err.Error())
		return err
	}

	if err := dbObj.Delete(ctx, userCred); err != nil {
		err := errors.Wrapf(err, "Delete")
		dbObj.SetStatus(userCred, api.ClusterResourceStatusDeleteFail, err.Error())
		return err
	}
	dbObj.PostDelete(ctx, userCred)
	*/
	return nil
}

func SyncUpdatedClusterResource(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	man IClusterModelManager,
	dbObj IClusterModel, extObj interface{}) error {
	if err := GetClusterResAPI().UpdateFromRemoteObject(dbObj, ctx, userCred, extObj); err != nil {
		return errors.Wrap(err, "SyncUpdatedClusterResource")
	}
	return nil
}

func (m *SClusterResourceBaseManager) ListRemoteObjects(clsCli *client.ClusterManager) ([]interface{}, error) {
	version, err := clsCli.GetClientset().Discovery().ServerVersion()
	if err != nil {
		return nil, err
	}

	resInfo := m.GetK8sResourceInfo(version)
	log.Infof("List remote object %v, server version %v.%v, resource gvr: %v, %v, %v",
		resInfo.KindName, version.Major, version.Minor, resInfo.Group, resInfo.Version, resInfo.ResourceName)

	// TODO: Use generic lister to replace it:)
	if resInfo.KindName == api.KindNameClusterRoleBinding || resInfo.KindName == api.KindNameRoleBinding {
		k8sCli := clsCli.GetHandler()
		objs, err := k8sCli.List(resInfo.ResourceName, "", labels.Everything().String())
		if err != nil {
			return nil, errors.Wrapf(err, "list k8s %s remote objects", resInfo.KindName)
		}
		ret := make([]interface{}, len(objs))
		for i := range objs {
			ret[i] = objs[i]
		}
		return ret, nil
	}

	cli := clsCli.GetClient()
	objs, err := cli.K8S().List(resInfo.Group, resInfo.Version, resInfo.KindName, "")
	if err != nil {
		return nil, errors.Wrapf(err, "list k8s %s remote objects", resInfo.ResourceName)
	}

	ret := make([]interface{}, len(objs))
	for i := range objs {
		newObj := resInfo.Object.DeepCopyObject()
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(objs[i].(*unstructured.Unstructured).Object, newObj); err != nil {
			return nil, errors.Wrap(err, "convert from unstructured")
		}
		ret[i] = newObj
	}
	return ret, nil
}

func (m *SClusterResourceBaseManager) GetRemoteObjectGlobalId(cluster *SCluster, obj interface{}) string {
	return string(obj.(metav1.Object).GetUID())
}

func (m *SClusterResourceBaseManager) IsRemoteObjectLocalExist(userCred mcclient.TokenCredential, cluster *SCluster, obj interface{}) (IClusterModel, bool, error) {
	metaObj := obj.(metav1.Object)
	localObj, err := m.GetByName(userCred, cluster.GetId(), metaObj.GetName())
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, errors.Wrapf(err, "get cluster %s %s/%s", cluster.GetName(), m.Keyword(), metaObj.GetName())
	}
	return localObj, true, nil
}

func (m *SClusterResourceBaseManager) NewFromRemoteObject(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	cluster *SCluster,
	obj interface{},
) (IClusterModel, error) {
	dbObj, err := db.NewModelObject(m)
	if err != nil {
		return nil, errors.Wrap(err, "NewModelObject")
	}
	metaObj := obj.(metav1.Object)
	dbObj.(db.IExternalizedModel).SetExternalId(m.GetRemoteObjectGlobalId(cluster, obj))
	dbObj.(IClusterModel).SetName(metaObj.GetName())
	dbObj.(IClusterModel).SetCluster(userCred, cluster)
	return dbObj.(IClusterModel), nil
}

func (obj *SClusterResourceBase) GetClusterId() string {
	return obj.ClusterId
}

func (obj *SClusterResourceBase) SetName(name string) {
	obj.Name = name
}

func (obj *SClusterResourceBase) UpdateFromRemoteObject(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	extObj interface{}) error {
	metaObj := extObj.(metav1.Object)
	resVersion := metaObj.GetResourceVersion()
	ver, _ := strconv.ParseInt(resVersion, 10, 32)
	var curResVer int64
	if obj.ResourceVersion != "" {
		curResVer, _ = strconv.ParseInt(obj.ResourceVersion, 10, 32)
	}
	if ver < curResVer {
		return errors.Errorf("remote object resourceVersion less than local: %d < %d", ver, curResVer)
	}
	if obj.GetName() != metaObj.GetName() {
		obj.SetName(metaObj.GetName())
	}
	if obj.ResourceVersion != resVersion {
		obj.ResourceVersion = resVersion
	}
	return nil
}

func (obj *SClusterResourceBase) SetStatusByRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, extObj interface{}) error {
	obj.Status = api.ClusterResourceStatusActive
	return nil
}

func (m *SClusterResourceBaseManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.ClusterResourceListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SStatusDomainLevelResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StatusDomainLevelResourceListInput)
	if err != nil {
		return nil, err
	}
	if input.Cluster != "" {
		cls, err := ClusterManager.FetchClusterByIdOrName(userCred, input.Cluster)
		if err != nil {
			return nil, err
		}
		input.Cluster = cls.GetId()
		q = q.Equals("cluster_id", cls.GetId())
	}
	return q, nil
}

func FetchClusterResourceCustomizeColumns(
	baseGet func(obj interface{}) interface{},
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []interface{} {
	ret := make([]interface{}, len(objs))
	for idx := range objs {
		obj := objs[idx].(IClusterModel)

		baseDetail := baseGet(obj)
		cli, err := obj.GetClusterClient()
		if err != nil {
			log.Errorf("get object %s cluster client error: %v", obj.Keyword(), err)
			ret[idx] = baseDetail
			continue
		}

		serverVersion, err := cli.GetClientset().Discovery().ServerVersion()
		if err != nil {
			return nil
		}

		k8sResInfo := obj.GetClusterModelManager().GetK8sResourceInfo(serverVersion)
		var k8sObj runtime.Object
		if k8sResInfo.Object != nil {
			k8sObj, err = GetK8sObject(obj)
			if err != nil {
				log.Errorf("get object from k8s error: %v", err)
				ret[idx] = baseDetail
				continue
			}
		}
		out := obj.GetDetails(cli, baseDetail, k8sObj, isList)
		ret[idx] = out
	}
	return ret
}

type iPurgeClusterResource interface {
	GetId() string
	RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error
}

func (m *SClusterResourceBaseManager) PurgeAllByCluster(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster) error {
	// objs := make([]IClusterModel, 0)
	objs := make([]interface{}, 0)
	q := m.Query().Equals("cluster_id", cluster.GetId())
	if err := db.FetchModelObjects(m, q, &objs); err != nil {
		return errors.Wrapf(err, "Fetch all %s objects when purge all", m.KeywordPlural())
	}
	for i := range objs {
		obj := objs[i]
		objPtr := GetObjectPtr(obj).(iPurgeClusterResource)
		if err := objPtr.RealDelete(ctx, userCred); err != nil {
			return errors.Wrapf(err, "delete %s object %s", m.Keyword(), objPtr.GetId())
		}
	}
	return nil
}

func (obj *SClusterResourceBase) GetDetails(
	cli *client.ClusterManager,
	base interface{},
	k8sObj runtime.Object,
	isList bool,
) interface{} {
	out := api.ClusterResourceDetail{
		StatusDomainLevelResourceDetails: base.(apis.StatusDomainLevelResourceDetails),
	}
	cls, err := obj.GetCluster()
	if err != nil {
		log.Errorf("Get resource %s cluster error: %v", obj.GetName(), err)
		return out
	}
	out.Cluster = cls.GetName()
	out.ClusterId = cls.GetId()
	out.ClusterID = cls.GetId()
	return GetK8SResourceMetaDetail(k8sObj, out)
}

func (m *SClusterResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []interface{} {
	baseGet := func(obj interface{}) interface{} {
		vRows := m.SStatusDomainLevelResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, []interface{}{obj}, fields, isList)
		return vRows[0]
	}
	return FetchClusterResourceCustomizeColumns(baseGet, ctx, userCred, query, objs, fields, isList)
}

func GetK8sObject(res IClusterModel) (runtime.Object, error) {
	cli, err := res.GetClusterClient()
	man := res.GetClusterModelManager()
	serverVersion, err := cli.GetClientset().Discovery().ServerVersion()
	if err != nil {
		return nil, err
	}
	info := man.GetK8sResourceInfo(serverVersion)
	if err != nil {
		return nil, errors.Wrapf(err, "get object %s/%s kubernetes client", info.ResourceName, res.GetName())
	}
	namespaceName := ""
	if nsResObj, ok := res.(INamespaceModel); ok {
		nsObj, err := nsResObj.GetNamespace()
		if err != nil {
			return nil, errors.Wrapf(err, "get object %s/%s local db namespace", info.ResourceName, res.GetName())
		}
		namespaceName = nsObj.GetName()
	}
	// k8sObj, err := cli.GetHandler().Get(info.ResourceName, namespaceName, res.GetName())
	k8sObj, err := cli.GetClient().K8S().Get(info.ResourceName, namespaceName, res.GetName())
	if err != nil {
		return nil, errors.Wrapf(err, "get object from k8s %s/%s/%s", info.ResourceName, namespaceName, res.GetName())
	}

	if unstruct, ok := k8sObj.(*unstructured.Unstructured); ok {
		newObj := info.Object.DeepCopyObject()
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstruct.Object, newObj); err != nil {
			return nil, errors.Wrap(err, "convert from unstructured")
		}
		k8sObj = newObj
	}

	return k8sObj, nil
}

func UpdateK8sObject(res IClusterModel, data jsonutils.JSONObject) (runtime.Object, error) {
	cli, err := res.GetClusterClient()
	if err != nil {
		return nil, errors.Wrap(err, "get cluster client")
	}
	handler := cli.GetHandler()
	serverVersion, err := cli.GetClientset().Discovery().ServerVersion()
	if err != nil {
		return nil, err
	}
	resInfo := res.GetClusterModelManager().GetK8sResourceInfo(serverVersion)
	rawStr, err := data.GetString()
	if err != nil {
		return nil, httperrors.NewInputParameterError("Get body raw data: %v", err)
	}
	namespaceName := ""
	if nsResObj, ok := res.(INamespaceModel); ok {
		nsObj, err := nsResObj.GetNamespace()
		if err != nil {
			return nil, errors.Wrapf(err, "get object %s/%s local db namespace", resInfo.ResourceName, res.GetName())
		}
		namespaceName = nsObj.GetName()
	}
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(rawStr), nil, nil)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Decode to runtime object error: %v", err)
	}
	putSpec := runtime.Unknown{}
	objStr, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	if err := json.NewDecoder(strings.NewReader(string(objStr))).Decode(&putSpec); err != nil {
		return nil, err
	}
	_, err = handler.Update(resInfo.ResourceName, namespaceName, res.GetName(), &putSpec)
	if err != nil {
		return nil, errors.Wrap(err, "update remote k8s object")
	}
	return GetK8sObject(res)
}

func GetK8SResourceMetaDetail(k8sObj runtime.Object, detail api.ClusterResourceDetail) api.ClusterResourceDetail {
	if k8sObj == nil {
		return detail
	}
	metaObj := k8sObj.(metav1.Object)
	detail.ClusterK8SResourceMetaDetail = &api.ClusterK8SResourceMetaDetail{
		TypeMeta:                   GetK8SObjectTypeMeta(k8sObj),
		ResourceVersion:            metaObj.GetResourceVersion(),
		Generation:                 metaObj.GetGeneration(),
		CreationTimestamp:          metaObj.GetCreationTimestamp().Time,
		DeletionGracePeriodSeconds: metaObj.GetDeletionGracePeriodSeconds(),
		Labels:                     metaObj.GetLabels(),
		Annotations:                metaObj.GetAnnotations(),
		OwnerReferences:            metaObj.GetOwnerReferences(),
		Finalizers:                 metaObj.GetFinalizers(),
	}
	if deletionTimestamp := metaObj.GetDeletionTimestamp(); deletionTimestamp != nil {
		detail.ClusterK8SResourceMetaDetail.DeletionTimestamp = &deletionTimestamp.Time
	}
	return detail
}

func (res *SClusterResourceBase) AllowGetDetailsRawdata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	//return res.AllowGetDetails()
	// TODO: use rbac to check
	return true
}

func (res *SClusterResourceBase) GetDetailsRawdata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	k8sObj, err := GetK8sObject(res)
	if err != nil {
		return nil, err
	}
	return K8SObjectToJSONObject(k8sObj), nil
}

func (res *SClusterResourceBase) AllowUpdateRawdata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return true
}

func (res *SClusterResourceBase) UpdateRawdata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	k8sObj, err := UpdateK8sObject(res, data)
	if err != nil {
		return nil, err
	}
	return K8SObjectToJSONObject(k8sObj), nil
}

// GetExtraDetails is deprecated
func (res *SClusterResourceBase) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (api.ClusterResourceDetail, error) {
	return api.ClusterResourceDetail{}, nil
}

func CreateRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, man IClusterModelManager, model IClusterModel, data jsonutils.JSONObject) (interface{}, error) {
	cli, err := model.GetClusterClient()
	if err != nil {
		return nil, errors.Wrap(err, "get cluster client")
	}
	obj, err := man.NewRemoteObjectForCreate(model, cli, data)
	if err != nil {
		return nil, errors.Wrap(err, "NewRemoteObjectForCreate")
	}
	obj, err = man.CreateRemoteObject(model, cli, obj)
	if err != nil {
		return nil, errors.Wrap(err, "CreateRemoteObject")
	}
	if err := GetClusterResAPI().UpdateFromRemoteObject(model, ctx, userCred, obj); err != nil {
		return nil, errors.Wrap(err, "UpdateFromRemoteObject after CreateRemoteObject")
	}
	return obj, nil
}

func UpdateRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, model IClusterModel, data jsonutils.JSONObject) (interface{}, error) {
	cli, err := model.GetClusterClient()
	if err != nil {
		return nil, errors.Wrap(err, "get cluster client")
	}
	extObj, err := model.GetRemoteObject()
	if err != nil {
		return nil, errors.Wrap(err, "get remote object")
	}
	extObj, err = model.NewRemoteObjectForUpdate(cli, extObj, data)
	if err != nil {
		return nil, errors.Wrap(err, "NewRemoteObjectForUpdate")
	}
	extObj, err = model.UpdateRemoteObject(extObj)
	if err != nil {
		return nil, errors.Wrap(err, "UpdateRemoteObject")
	}
	extObj, err = model.GetRemoteObject()
	if err != nil {
		return nil, errors.Wrap(err, "get remote object after updated")
	}
	if err := GetClusterResAPI().UpdateFromRemoteObject(model, ctx, userCred, extObj); err != nil {
		return nil, errors.Wrap(err, "UpdateFromRemoteObject")
	}
	return extObj, nil
}

func (m *SClusterResourceBaseManager) NewRemoteObjectForCreate(_ IClusterModel, _ *client.ClusterManager, data jsonutils.JSONObject) (interface{}, error) {
	return nil, fmt.Errorf("NewRemoteObjectForCreate of %s not override", m.kindName)
}

func (m *SClusterResourceBaseManager) CreateRemoteObject(_ IClusterModel, cli *client.ClusterManager, obj interface{}) (interface{}, error) {
	metaObj := obj.(metav1.Object)
	return cli.GetHandler().CreateV2(m.resourceName, metaObj.GetNamespace(), obj.(runtime.Object))
}

func DeleteRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, man IClusterModelManager, model IClusterModel, data jsonutils.JSONObject) error {
	if err := model.DeleteRemoteObject(); err != nil {
		return errors.Wrap(err, "DeleteRemoteObject")
	}
	return nil
}

func (m *SClusterResourceBaseManager) GetClusterModelManager() IClusterModelManager {
	return m.GetIModelManager().(IClusterModelManager)
}

func (obj *SClusterResourceBase) LogPrefix() string {
	return fmt.Sprintf("cluster %s %s %s(%s)", obj.ClusterId, obj.Keyword(), obj.GetName(), obj.GetId())
}

func (obj *SClusterResourceBase) GetClusterModelManager() IClusterModelManager {
	return obj.GetModelManager().(IClusterModelManager)
}

func (obj *SClusterResourceBase) GetRemoteObject() (interface{}, error) {
	cli, err := obj.GetClusterClient()
	if err != nil {
		return nil, errors.Wrapf(err, "get %s/%s cluster client", obj.Keyword(), obj.GetName())
	}
	serverVersion, err := cli.GetClientset().Discovery().ServerVersion()
	if err != nil {
		return nil, err
	}
	resInfo := obj.GetClusterModelManager().GetK8sResourceInfo(serverVersion)
	k8sCli := cli.GetHandler()
	return k8sCli.Get(resInfo.ResourceName, "", obj.GetName())
}

func (obj *SClusterResourceBase) GetK8sObject() (runtime.Object, error) {
	ret, err := obj.GetRemoteObject()
	if err != nil {
		return nil, err
	}
	return ret.(runtime.Object), nil
}

func (obj *SClusterResourceBase) UpdateRemoteObject(remoteObj interface{}) (interface{}, error) {
	cli, err := obj.GetClusterClient()
	if err != nil {
		return nil, errors.Wrapf(err, "get %s/%s cluster client", obj.Keyword(), obj.GetName())
	}
	serverVersion, err := cli.GetClientset().Discovery().ServerVersion()
	if err != nil {
		return nil, err
	}
	resInfo := obj.GetClusterModelManager().GetK8sResourceInfo(serverVersion)
	k8sCli := cli.GetHandler()
	return k8sCli.UpdateV2(resInfo.ResourceName, remoteObj.(runtime.Object))
}

func (res *SClusterResourceBase) NewRemoteObjectForUpdate(cli *client.ClusterManager, remoteObj interface{}, data jsonutils.JSONObject) (interface{}, error) {
	return remoteObj, nil
}

func (obj *SClusterResourceBase) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("Resource %s delete do nothing", obj.Keyword())
	return nil
}

func (obj *SClusterResourceBase) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return obj.SStatusDomainLevelResourceBase.Delete(ctx, userCred)
}

func (obj *SClusterResourceBase) DeleteRemoteObject() error {
	cli, err := obj.GetClusterClient()
	if err != nil {
		return errors.Wrapf(err, "get %s/%s cluster client", obj.Keyword(), obj.GetName())
	}
	serverVersion, err := cli.GetClientset().Discovery().ServerVersion()
	if err != nil {
		return err
	}
	resInfo := obj.GetClusterModelManager().GetK8sResourceInfo(serverVersion)
	if err := cli.GetHandler().Delete(resInfo.ResourceName, "", obj.GetName(), &metav1.DeleteOptions{}); err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

func (_ *SClusterResourceBase) StartCreateTask(resObj IClusterModel, ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, data *jsonutils.JSONDict, parentId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ClusterResourceCreateTask", resObj, userCred, data, parentId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (obj *SClusterResourceBase) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.ClusterResourceUpdateInput) (*api.ClusterResourceUpdateInput, error) {
	if input.Name != "" {
		return nil, httperrors.NewInputParameterError("can not update cluster resource name")
	}
	bInput, err := obj.SStatusDomainLevelResourceBase.ValidateUpdateData(ctx, userCred, query, input.StatusDomainLevelResourceBaseUpdateInput)
	if err != nil {
		return nil, err
	}
	input.StatusDomainLevelResourceBaseUpdateInput = bInput
	return input, nil
}

func (_ *SClusterResourceBase) StartUpdateTask(resObj IClusterModel, ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ClusterResourceUpdateTask", resObj, userCred, data, parentId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (_ *SClusterResourceBase) StartDeleteTask(resObj IClusterModel, ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ClusterResourceDeleteTask", resObj, userCred, data, parentId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

// OnRemoteObjectCreate invoked when remote object created
func (m *SClusterResourceBaseManager) OnRemoteObjectCreate(ctx context.Context, userCred mcclient.TokenCredential, cluster manager.ICluster, resMan manager.IK8sResourceManager, obj runtime.Object) {
	// log.Debugf("Remote object create: %s", m.kObjLogPrefix(cluster, obj))
	if err := processCreateOrUpdateByRemoteObject(ctx, userCred, cluster, resMan.(IClusterModelManager), obj); err != nil {
		log.Errorf("OnRemoteObjectCreate %s error: %v", m.kObjLogPrefix(cluster, obj), err)
	}
}

// OnRemoteObjectUpdate invoked when remote resource updated
func (m *SClusterResourceBaseManager) OnRemoteObjectUpdate(ctx context.Context, userCred mcclient.TokenCredential, cluster manager.ICluster, resMan manager.IK8sResourceManager, _, newObj runtime.Object) {
	// log.Debugf("Remote object update: %s", m.kObjLogPrefix(cluster, newObj))
	if err := processCreateOrUpdateByRemoteObject(ctx, userCred, cluster, resMan.(IClusterModelManager), newObj); err != nil {
		log.Errorf("OnRemoteObjectUpdate %s error: %v", m.kObjLogPrefix(cluster, newObj), err)
	}
}

// OnRemoteObjectDelete invoked when remote resource deleted
func (m *SClusterResourceBaseManager) OnRemoteObjectDelete(ctx context.Context, userCred mcclient.TokenCredential, cluster manager.ICluster, resMan manager.IK8sResourceManager, obj runtime.Object) {
	// log.Debugf("Remote object delete: %s", m.kObjLogPrefix(cluster, obj))
	if err := processRemoteObjectDelete(ctx, userCred, cluster, resMan.(IClusterModelManager), obj); err != nil {
		log.Errorf("processRemoteObjectDelete %s error: %v", m.kObjLogPrefix(cluster, obj), err)
	}
}

func processCreateOrUpdateByRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, cluster manager.ICluster, resMan IClusterModelManager, obj runtime.Object) error {
	metaObj := obj.(metav1.Object)
	objName := metaObj.GetName()
	dbObj, exist, err := resMan.IsRemoteObjectLocalExist(userCred, cluster.(*SCluster), obj)
	if err != nil {
		log.Errorf("Create or update by cluster %q remote object %s/%s/%s error: %v", cluster.GetName(), resMan.Keyword(), metaObj.GetNamespace(), objName, err)
		return errors.Wrap(err, "check remote object is local exist")
	}

	if exist {
		lockman.LockObject(ctx, dbObj)
		defer lockman.ReleaseObject(ctx, dbObj)

		if err := onRemoteObjectUpdate(resMan, ctx, userCred, dbObj, obj); err != nil {
			return errors.Wrap(err, "call onRemoteObjectUpdate")
		}
		return nil
	} else {
		lockman.LockClass(ctx, resMan, db.GetLockClassKey(resMan, userCred))
		defer lockman.ReleaseClass(ctx, resMan, db.GetLockClassKey(resMan, userCred))

		if err := onRemoteObjectCreate(resMan, ctx, userCred, cluster, obj); err != nil {
			return errors.Wrap(err, "call onRemoteObjectCreate")
		}
		return nil
	}
}

func processRemoteObjectDelete(ctx context.Context, userCred mcclient.TokenCredential, cluster manager.ICluster, resMan IClusterModelManager, obj runtime.Object) error {
	metaObj := obj.(metav1.Object)
	objName := metaObj.GetName()
	objNamespace := metaObj.GetNamespace()
	isNamespace := resMan.IsNamespaceScope()
	dbNsId := ""
	if isNamespace {
		dbNs, err := GetNamespaceManager().GetByName(userCred, cluster.GetId(), objNamespace)
		if err != nil {
			return errors.Wrapf(err, "OnRemoteObjectDelete for cluster %s %s get namespace", cluster.GetName(), resMan.Keyword())
		}
		dbNsId = dbNs.GetId()
	}

	dbObj, err := FetchClusterResourceByName(resMan, userCred, cluster.GetId(), dbNsId, objName)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			// local object already deleted
			return nil
		}
		return errors.Wrapf(err, "FetchClusterResourceByName cluster %s %s %s/%s", cluster.GetName(), resMan.Keyword(), dbNsId, objName)
	}

	lockman.LockObject(ctx, dbObj)
	defer lockman.ReleaseObject(ctx, dbObj)

	return OnRemoteObjectDelete(resMan, ctx, userCred, dbObj)
}

func (m *SClusterResourceBaseManager) kObjLogPrefix(cluster manager.ICluster, obj runtime.Object) string {
	kind := obj.GetObjectKind().GroupVersionKind().String()
	kObj := obj.(metav1.Object)
	return fmt.Sprintf("cluster %s(%s) remote object %s/%s/%s", cluster.GetName(), cluster.GetId(), kind, kObj.GetNamespace(), kObj.GetName())
}

func onRemoteObjectCreate(resMan IClusterModelManager, ctx context.Context, userCred mcclient.TokenCredential, cluster manager.ICluster, obj runtime.Object) error {
	objName := obj.(metav1.Object).GetName()
	// create localObj
	log.Debugf("cluster %s remote object %s/%s created, sync to local", cluster.GetName(), resMan.Keyword(), objName)
	if _, err := NewFromRemoteObject(ctx, userCred, resMan, cluster.(*SCluster), obj); err != nil {
		return errors.Wrapf(err, "NewFromRemoteObject for %s", resMan.Keyword())
	}
	return nil
}

func onRemoteObjectUpdate(resMan IClusterModelManager, ctx context.Context, userCred mcclient.TokenCredential, dbObj IClusterModel, newObj runtime.Object) error {
	// log.Debugf("remote object %s/%s update, sync to local", resMan.Keyword(), dbObj.GetName())
	if err := SyncUpdatedClusterResource(ctx, userCred, resMan, dbObj, newObj); err != nil {
		return errors.Wrapf(err, "onRemoteObjectUpdate SyncUpdatedClusterResource %s", dbObj.LogPrefix())
	}
	return nil
}

func OnRemoteObjectDelete(resMan IClusterModelManager, ctx context.Context, userCred mcclient.TokenCredential, dbObj IClusterModel) error {
	// log.Debugf("remote object %s/%s deleted, delete local", resMan.Keyword(), dbObj.GetName())
	if err := SyncRemovedClusterResource(ctx, userCred, dbObj); err != nil {
		return errors.Wrapf(err, "OnRemoteObjectDelete %s %s SyncRemovedClusterResource error: %v", resMan.Keyword(), dbObj.GetName())
	}
	return nil
}

func GetResourcesByClusters(man IClusterModelManager, clusterIds []string, ret interface{}) error {
	q := man.Query().In("cluster_id", clusterIds)
	if err := q.All(ret); err != nil {
		return errors.Wrapf(err, "fetch %s resources by clusters")
	}
	return nil
}

func ValidateUpdateData(obj IClusterModel, ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if name, _ := data.GetString("name"); name != "" {
		return nil, httperrors.NewInputParameterError("can not update cluster resource name")
	}
	cli, err := obj.GetClusterClient()
	if err != nil {
		return nil, errors.Wrapf(err, "get resource %s/%s cluster client", obj.Keyword(), obj.GetName())
	}
	extObj, err := obj.GetRemoteObject()
	if err != nil {
		return nil, errors.Wrapf(err, "get resource %s/%s remote object", obj.Keyword(), obj.GetName())
	}
	extObj, err = obj.NewRemoteObjectForUpdate(cli, extObj, data)
	if err != nil {
		return nil, errors.Wrapf(err, "NewRemoteObjectForUpdate for resource %s/%s", obj.Keyword(), obj.GetName())
	}
	return data, nil
}
