package models

import (
	"context"
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/utils/logclient"
)

var (
	fedResAPI IFedResAPI
)

func init() {
	GetFedResAPI()
}

func GetFedResAPI() IFedResAPI {
	if fedResAPI == nil {
		fedResAPI = newFedResAPI()
	}
	return fedResAPI
}

type IFedResAPI interface {
	ClusterScope() IFedClusterResAPI
	NamespaceScope() IFedNamespaceResAPI
	JointResAPI() IFedJointResAPI

	// GetJointModel fetch federated related joint object
	GetJointModel(obj IFedModel, clusterId string) (IFedJointClusterModel, error)
	// IsAttach2Cluster check federated object is attach to specified cluster
	IsAttach2Cluster(obj IFedModel, clusterId string) (bool, error)
	// GetAttachedClusters fetch clusters attached to current federated object
	GetAttachedClusters(obj IFedModel) ([]SCluster, error)

	// PerformAttachCluster sync federated template object to cluster
	PerformAttachCluster(obj IFedModel, ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) (IFedJointClusterModel, error)

	// JustAttachCluster just create joint model
	JustAttachCluster(obj IFedModel, ctx context.Context, userCred mcclient.TokenCredential, clusterId string) (IFedJointClusterModel, error)

	// PerformSyncCluster sync resource to cluster
	PerformSyncCluster(obj IFedModel, ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) error

	// PerformDetachCluster delete federated releated object inside cluster
	PerformDetachCluster(obj IFedModel, ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) error

	// StartUpdateTask called when federated object post update
	StartUpdateTask(obj IFedModel, ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error
	// StartSyncTask sync federated object current template to attached clusters
	StartSyncTask(obj IFedModel, ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error
}

type sFedResAPI struct {
	clusterScope   IFedClusterResAPI
	namespaceScope IFedNamespaceResAPI
	jointResAPI    IFedJointResAPI
}

func newFedResAPI() IFedResAPI {
	a := new(sFedResAPI)
	clusterScope := newFedClusterResAPI()
	namespaceScope := newFedNamespaceResAPI()
	jointResAPI := newFedJointResAPI()
	a.clusterScope = clusterScope
	a.namespaceScope = namespaceScope
	a.jointResAPI = jointResAPI
	return a
}

func (a sFedResAPI) ClusterScope() IFedClusterResAPI {
	return a.clusterScope
}

func (a sFedResAPI) NamespaceScope() IFedNamespaceResAPI {
	return a.namespaceScope
}

func (a sFedResAPI) JointResAPI() IFedJointResAPI {
	return a.jointResAPI
}

func (a sFedResAPI) GetJointModel(obj IFedModel, clusterId string) (IFedJointClusterModel, error) {
	jMan := obj.GetJointModelManager()
	jObj, err := GetFederatedJointClusterModel(jMan, obj.GetId(), clusterId)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return nil, err
	}
	return jObj, nil
}

func (a sFedResAPI) IsAttach2Cluster(obj IFedModel, clusterId string) (bool, error) {
	jObj, err := a.GetJointModel(obj, clusterId)
	if err != nil {
		return false, err
	}
	return jObj != nil, nil
}

func (a sFedResAPI) GetAttachedClusters(obj IFedModel) ([]SCluster, error) {
	jm := obj.GetJointModelManager()
	clusters := make([]SCluster, 0)
	q := GetClusterManager().Query()
	sq := jm.Query("cluster_id").Equals("federatedresource_id", obj.GetId()).SubQuery()
	q = q.In("id", sq)
	if err := db.FetchModelObjects(GetClusterManager(), q, &clusters); err != nil {
		return nil, errors.Wrapf(err, "get federated resource %s %s attached clusters", obj.Keyword(), obj.GetName())
	}
	return clusters, nil
}

func (a sFedResAPI) StartUpdateTask(obj IFedModel, ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "FedResourceUpdateTask", obj, userCred, data, parentTaskId, "")
	if err != nil {
		return errors.Wrap(err, "New FedResourceUpdateTask")
	}
	task.ScheduleRun(nil)
	return nil
}

func (a sFedResAPI) StartSyncTask(obj IFedModel, ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "FedResourceSyncTask", obj, userCred, data, parentTaskId, "")
	if err != nil {
		return errors.Wrap(err, "New FedResourceSyncTask")
	}
	task.ScheduleRun(nil)
	return nil
}

func (a sFedResAPI) PerformSyncCluster(obj IFedModel, ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) error {
	jObj, data, err := obj.ValidateJointCluster(ctx, userCred, data)
	if err != nil {
		return err
	}
	if err := a.performSyncCluster(jObj, ctx, userCred); err != nil {
		logclient.LogWithContext(ctx, obj, logclient.ActionResourceSync, err, userCred, false)
		return err
	}
	logclient.LogWithContext(ctx, obj, logclient.ActionResourceSync, data, userCred, true)
	return nil
}

func (a sFedResAPI) performSyncCluster(jObj IFedJointClusterModel, ctx context.Context, userCred mcclient.TokenCredential) error {
	return a.jointResAPI.ReconcileResource(jObj, ctx, userCred)
}

func (a sFedResAPI) PerformAttachCluster(obj IFedModel, ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) (IFedJointClusterModel, error) {
	data, err := obj.ValidateAttachCluster(ctx, userCred, data)
	if err != nil {
		return nil, err
	}
	clusterId, err := data.GetString("cluster_id")
	if err != nil {
		return nil, err
	}
	jObj, err := a.JustAttachCluster(obj, ctx, userCred, clusterId)
	if err != nil {
		logclient.LogWithContext(ctx, obj, logclient.ActionResourceAttach, err, userCred, false)
		return nil, err
	}
	logclient.LogWithContext(ctx, obj, logclient.ActionResourceAttach, data, userCred, true)

	if err := a.PerformSyncCluster(obj, ctx, userCred, data); err != nil {
		return nil, err
	}
	return jObj, nil
}

func (a sFedResAPI) JustAttachCluster(obj IFedModel, ctx context.Context, userCred mcclient.TokenCredential, clusterId string) (IFedJointClusterModel, error) {
	defer lockman.ReleaseObject(ctx, obj)
	lockman.LockObject(ctx, obj)

	cls, err := GetClusterManager().GetCluster(clusterId)
	if err != nil {
		return nil, errors.Wrapf(err, "get cluster %s", clusterId)
	}

	attached, err := a.IsAttach2Cluster(obj, clusterId)
	if err != nil {
		return nil, errors.Wrap(err, "check IsAttach2Cluster")
	}
	if attached {
		return nil, errors.Errorf("%s %s has been attached to cluster %s", obj.Keyword(), obj.GetId(), clusterId)
	}
	jointMan := obj.GetJointModelManager()
	jointModel, err := db.NewModelObject(jointMan)
	if err != nil {
		return nil, errors.Wrapf(err, "new joint model %s", jointMan.Keyword())
	}
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(obj.GetId()), jointMan.GetMasterFieldName())
	data.Add(jsonutils.NewString(clusterId), jointMan.GetSlaveFieldName())
	if err := data.Unmarshal(jointModel); err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if err := jointMan.TableSpec().Insert(ctx, jointModel); err != nil {
		return nil, errors.Wrap(err, "insert joint model")
	}
	db.OpsLog.LogAttachEvent(ctx, obj, cls, userCred, nil)
	return jointModel.(IFedJointClusterModel), nil
}

func (a sFedResAPI) PerformDetachCluster(obj IFedModel, ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) error {
	data, err := obj.ValidateDetachCluster(ctx, userCred, data)
	if err != nil {
		return err
	}
	clusterId, _ := data.GetString("cluster_id")
	if err := a.detachCluster(obj, ctx, userCred, clusterId); err != nil {
		logclient.LogWithContext(ctx, obj, logclient.ActionResourceDetach, err, userCred, false)
		return err
	}
	logclient.LogWithContext(ctx, obj, logclient.ActionResourceDetach, data, userCred, true)
	return nil
}

func (a sFedResAPI) detachCluster(obj IFedModel, ctx context.Context, userCred mcclient.TokenCredential, clusterId string) error {
	defer lockman.ReleaseObject(ctx, obj)
	lockman.LockObject(ctx, obj)

	attached, err := a.IsAttach2Cluster(obj, clusterId)
	if err != nil {
		return errors.Wrap(err, "check IsAttach2Cluster")
	}
	if !attached {
		return nil
	}

	jointModel, err := a.GetJointModel(obj, clusterId)
	if err != nil {
		return errors.Wrap(err, "detach get joint model")
	}

	// TODO: start task todo it
	return jointModel.Detach(ctx, userCred)
}

type IFedClusterResAPI interface {
}

type sFedClusterResAPI struct{}

func newFedClusterResAPI() IFedClusterResAPI {
	return &sFedClusterResAPI{}
}

type IFedNamespaceResAPI interface {
	// StartAttachClusterTask start background task to attach fedreated namespace resource to cluster
	StartAttachClusterTask(obj IFedNamespaceModel, ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error
}

type sFedNamespaceResAPI struct{}

func newFedNamespaceResAPI() IFedNamespaceResAPI {
	return &sFedNamespaceResAPI{}
}

func (a sFedNamespaceResAPI) StartAttachClusterTask(obj IFedNamespaceModel, ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "FedResourceAttachClusterTask", obj, userCred, data, parentTaskId, "")
	if err != nil {
		return errors.Wrap(err, "New FedResourceAttachClusterTask")
	}
	task.ScheduleRun(nil)
	return nil
}

type IFedJointResAPI interface {
	ClusterScope() IFedJointClusterResAPI
	NamespaceScope() IFedNamespaceJointClusterResAPI

	// IsResourceExist check joint federated object's resource whether exists in target cluster
	IsResourceExist(jObj IFedJointClusterModel, userCred mcclient.TokenCredential) (IClusterModel, bool, error)
	// ReconcileResource reconcile federated object to cluster
	ReconcileResource(jObj IFedJointClusterModel, ctx context.Context, userCred mcclient.TokenCredential) error
	// FetchResourceModel get joint object releated cluster object
	FetchResourceModel(jObj IFedJointClusterModel) (IClusterModel, error)
	// FetchFedResourceModel get joint object related master fedreated db object
	FetchFedResourceModel(jObj IFedJointClusterModel) (IFedModel, error)
	// IsNamespaceScope mark object is namespace scope
	IsNamespaceScope(jObj IFedJointClusterModel) bool
	// GetDetails get joint object details
	GetDetails(ctx context.Context, jObj IFedJointClusterModel, userCred mcclient.TokenCredential, base apis.JointResourceBaseDetails, isList bool) interface{}
}

type sFedJointResAPI struct {
	clusterScope   IFedJointClusterResAPI
	namespaceScope IFedNamespaceJointClusterResAPI
}

func newFedJointResAPI() IFedJointResAPI {
	clsScope := newFedJointClusterResAPI()
	a := new(sFedJointResAPI)
	a.clusterScope = clsScope
	a.namespaceScope = newFedNamespaceJointClusterResAPI(a)
	return a
}

func (a sFedJointResAPI) ClusterScope() IFedJointClusterResAPI {
	return a.clusterScope
}

func (a sFedJointResAPI) NamespaceScope() IFedNamespaceJointClusterResAPI {
	return a.namespaceScope
}

func (a sFedJointResAPI) IsNamespaceScope(jObj IFedJointClusterModel) bool {
	return jObj.GetResourceManager().IsNamespaceScope()
}

func (a sFedJointResAPI) FetchResourceModel(jObj IFedJointClusterModel) (IClusterModel, error) {
	man := jObj.GetResourceManager()
	// namespace scope resource should also fetched by resourceId
	return FetchClusterResourceById(man, jObj.GetClusterId(), "", jObj.GetResourceId())
}

func (a sFedJointResAPI) GetDetails(ctx context.Context, jObj IFedJointClusterModel, userCred mcclient.TokenCredential, base apis.JointResourceBaseDetails, isList bool) interface{} {
	out := api.FedJointClusterResourceDetails{
		JointResourceBaseDetails: base,
	}
	cluster, err := jObj.GetCluster()
	if err != nil {
		log.Errorf("get cluster %s object error: %v", jObj.GetClusterId(), err)
	} else {
		out.Cluster = cluster.GetName()
	}

	if fedObj, err := a.FetchFedResourceModel(jObj); err != nil {
		log.Errorf("get federated resource %s object error: %v", jObj.GetFedResourceId(), err)
	} else {
		out.FederatedResource = fedObj.GetName()
		out.FederatedResourceKeyword = fedObj.Keyword()
	}
	if a.IsNamespaceScope(jObj) {
		nsObj, err := a.namespaceScope.FetchClusterNamespace(userCred, jObj, cluster)
		if err == nil && nsObj != nil {
			out.Namespace = nsObj.GetName()
			out.NamespaceId = nsObj.GetId()
		}
	}
	if jObj.GetResourceId() != "" {
		resObj, err := a.FetchResourceModel(jObj)
		if err == nil && resObj != nil {
			out.Resource = resObj.GetName()
			out.ResourceStatus = resObj.GetStatus()
			out.ResourceKeyword = resObj.Keyword()
		}
	} else {
		out.ResourceStatus = api.FederatedResourceStatusNotBind
	}
	return out
}

func (a sFedJointResAPI) FetchResourceModelByName(jObj IFedJointClusterModel, userCred mcclient.TokenCredential) (IClusterModel, error) {
	cluster, err := jObj.GetCluster()
	if err != nil {
		return nil, errors.Wrapf(err, "get %s joint object cluster", jObj.Keyword())
	}
	fedObj, err := a.FetchFedResourceModel(jObj)
	if err != nil {
		return nil, errors.Wrapf(err, "get %s related federated resource", jObj.Keyword())
	}
	clsNsId := ""
	checkResName := fedObj.GetName()
	if a.IsNamespaceScope(jObj) {
		nsObj, err := a.namespaceScope.FetchClusterNamespace(userCred, jObj, cluster)
		if err != nil {
			return nil, errors.Wrapf(err, "get %s cluster %s namespace", cluster.GetName(), jObj.Keyword())
		}
		clsNsId = nsObj.GetId()
	}
	man := jObj.GetResourceManager()
	resObj, err := FetchClusterResourceByIdOrName(man, userCred, cluster.GetId(), clsNsId, checkResName)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "get %s cluster %s resource %s", jObj.Keyword(), cluster.GetName(), checkResName)
	}
	if resObj != nil {
		// cluster resource object already exists
		return resObj, nil
	}
	return nil, nil
}

func (a sFedJointResAPI) IsResourceExist(jObj IFedJointClusterModel, userCred mcclient.TokenCredential) (IClusterModel, bool, error) {
	if jObj.GetResourceId() == "" {
		// check cluster related same name resource whether exists
		resObj, err := a.FetchResourceModelByName(jObj, userCred)
		if err != nil {
			return nil, false, errors.Wrapf(err, "fetch joint %s resource model by name", jObj.Keyword())
		}
		if resObj != nil {
			// cluster resource object already exists
			return resObj, true, nil
		}
		return nil, false, nil
	}
	resObj, err := a.FetchResourceModel(jObj)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, errors.Wrapf(err, "FetchResourceModel of %s", jObj.Keyword())
	}
	if resObj == nil {
		return nil, false, nil
	}
	return resObj, true, nil
}

func (a sFedJointResAPI) ReconcileResource(jObj IFedJointClusterModel, ctx context.Context, userCred mcclient.TokenCredential) error {
	resObj, exist, err := a.IsResourceExist(jObj, userCred)
	if err != nil {
		return errors.Wrapf(err, "Check %s/%s cluster resource exist", jObj.Keyword(), jObj.GetName())
	}
	cluster, err := jObj.GetCluster()
	if err != nil {
		return errors.Wrap(err, "get joint object cluster")
	}
	ownerId := cluster.GetOwnerId()
	fedObj := db.JointMaster(jObj).(IFedModel)
	if exist {
		if err := a.updateResource(ctx, userCred, jObj, resObj); err != nil {
			return errors.Wrap(err, "UpdateClusterResource")
		}
		return nil
	}
	if err := a.createResource(ctx, userCred, ownerId, jObj, fedObj, cluster); err != nil {
		return errors.Wrap(err, "CreateClusterResource")
	}
	return nil
}

func (a sFedJointResAPI) createResource(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	jObj IFedJointClusterModel,
	fedObj IFedModel,
	cluster *SCluster,
) error {
	baseInput := new(api.NamespaceResourceCreateInput)
	baseInput.Name = fedObj.GetName()
	baseInput.ClusterId = cluster.GetId()
	baseInput.ProjectDomainId = cluster.DomainId
	if a.IsNamespaceScope(jObj) {
		clsNs, err := a.namespaceScope.FetchClusterNamespace(userCred, jObj, cluster)
		if err != nil {
			return errors.Wrapf(err, "get %s cluster %s namespace", jObj.Keyword(), cluster.GetName())
		}
		baseInput.NamespaceId = clsNs.GetId()
	}
	fedObj, err := GetFedResAPI().JointResAPI().FetchFedResourceModel(jObj)
	if err != nil {
		return errors.Wrapf(err, "get fed joint resource %s master object", jObj.Keyword())
	}
	data, err := jObj.GetResourceCreateData(ctx, userCred, fedObj, *baseInput)
	if err != nil {
		return errors.Wrapf(err, "get fed joint resource %s create to cluster %s resource data", jObj.Keyword(), cluster.GetName())
	}
	resObj, err := db.DoCreate(jObj.GetResourceManager(), ctx, userCred, nil, data, ownerId)
	if err != nil {
		return errors.Wrapf(err, "create cluster %q %s local resource object, data: %s", cluster.GetName(), jObj.GetResourceManager().Keyword(), data)
	}
	if err := jObj.SetResource(resObj.(IClusterModel)); err != nil {
		return errors.Wrapf(err, "set %s resource object", jObj.Keyword())
	}
	func() {
		lockman.LockObject(ctx, resObj)
		defer lockman.ReleaseObject(ctx, resObj)

		resObj.PostCreate(ctx, userCred, ownerId, nil, data)
	}()
	return nil
}

func (a sFedJointResAPI) updateResource(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	jObj IFedJointClusterModel,
	resObj IClusterModel,
) error {
	if err := jObj.SetResource(resObj.(IClusterModel)); err != nil {
		return errors.Wrapf(err, "set %s resource object", jObj.Keyword())
	}
	baseInput := new(api.NamespaceResourceUpdateInput)

	fedObj, err := GetFedResAPI().JointResAPI().FetchFedResourceModel(jObj)
	if err != nil {
		return errors.Wrapf(err, "get fed joint resource %s master object", jObj.Keyword())
	}
	data, err := jObj.GetResourceUpdateData(ctx, userCred, fedObj, resObj, *baseInput)
	if err != nil {
		return errors.Wrapf(err, "get fed joint resource %s update to cluster %s resource data", jObj.Keyword(), resObj.GetClusterId())
	}

	resObj.PostUpdate(ctx, userCred, nil, data)
	return nil
}

func (_ sFedJointResAPI) FetchFedResourceModel(jObj IFedJointClusterModel) (IFedModel, error) {
	fedMan := jObj.GetManager().GetFedManager()
	fObj, err := fedMan.FetchById(jObj.GetFedResourceId())
	if err != nil {
		return nil, errors.Wrapf(err, "get federated resource %s by id %s", fedMan.Keyword(), jObj.GetFedResourceId())
	}
	return fObj.(IFedModel), nil
}

type IFedJointClusterResAPI interface {
}

type fedJointClusterResAPI struct{}

func newFedJointClusterResAPI() IFedJointClusterResAPI {
	return &fedJointClusterResAPI{}
}

type IFedNamespaceJointClusterResAPI interface {
	FetchFedNamespace(jObj IFedNamespaceJointClusterModel) (*SFedNamespace, error)
	FetchClusterNamespace(userCred mcclient.TokenCredential, jObj IFedNamespaceJointClusterModel, cluster *SCluster) (*SNamespace, error)
	FetchModelsByFednamespace(man IFedNamespaceJointClusterManager, fednsId string) ([]IFedNamespaceJointClusterModel, error)
}

type fedNamespaceJointClusterResAPI struct {
	jointResAPI IFedJointResAPI
}

func newFedNamespaceJointClusterResAPI(
	jointResAPI IFedJointResAPI,
) IFedNamespaceJointClusterResAPI {
	return &fedNamespaceJointClusterResAPI{
		jointResAPI: jointResAPI,
	}
}

func (a fedNamespaceJointClusterResAPI) FetchFedNamespace(jObj IFedNamespaceJointClusterModel) (*SFedNamespace, error) {
	fedObj, err := a.jointResAPI.FetchFedResourceModel(jObj)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get fed %s obj", jObj.GetModelManager().Keyword())
	}
	return fedObj.(IFedNamespaceModel).GetFedNamespace()
}

func (a fedNamespaceJointClusterResAPI) FetchClusterNamespace(userCred mcclient.TokenCredential, jObj IFedNamespaceJointClusterModel, cluster *SCluster) (*SNamespace, error) {
	fedNs, err := a.FetchFedNamespace(jObj)
	if err != nil {
		return nil, errors.Wrapf(err, "get %s federatednamespace", jObj.Keyword())
	}
	nsName := fedNs.GetName()
	nsObj, err := GetNamespaceManager().GetByIdOrName(userCred, cluster.GetId(), nsName)
	if err != nil {
		return nil, errors.Wrapf(err, "get cluster %s namespace %s", cluster.GetName(), nsName)
	}
	return nsObj.(*SNamespace), nil
}

func (_ fedNamespaceJointClusterResAPI) FetchModelsByFednamespace(m IFedNamespaceJointClusterManager, fednsId string) ([]IFedNamespaceJointClusterModel, error) {
	objs := make([]interface{}, 0)
	sq := m.GetFedManager().Query("id").Equals("federatednamespace_id", fednsId).SubQuery()
	q := m.Query().In("federatedresource_id", sq)
	if err := db.FetchModelObjects(m, q, &objs); err != nil {
		return nil, err
	}
	ret := make([]IFedNamespaceJointClusterModel, len(objs))
	for i := range objs {
		obj := objs[i]
		objPtr := GetObjectPtr(obj).(IFedNamespaceJointClusterModel)
		ret[i] = objPtr
	}
	return ret, nil
}
