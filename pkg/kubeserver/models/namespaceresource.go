package models

import (
	"context"
	"database/sql"
	"fmt"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
)

// +onecloud:swagger-gen-ignore
type SNamespaceResourceBaseManager struct {
	SClusterResourceBaseManager
}

type SNamespaceResourceBase struct {
	SClusterResourceBase

	NamespaceId string `width:"36" charset:"ascii" nullable:"false" index:"true" list:"user"`
}

func NewNamespaceResourceBaseManager(
	dt interface{},
	tableName string,
	keyword string,
	keywordPlural string,
	resName string,
	groupName string,
	versionName string,
	kind string,
	object runtime.Object) SNamespaceResourceBaseManager {
	return SNamespaceResourceBaseManager{
		SClusterResourceBaseManager: NewClusterResourceBaseManager(dt, tableName, keyword, keywordPlural, resName, groupName, versionName, kind, object),
	}
}

func (r *SNamespaceResourceBase) GetUniqValues() jsonutils.JSONObject {
	return jsonutils.Marshal(map[string]string{
		"namespace_id": r.NamespaceId,
	})
}

func (m *SNamespaceResourceBaseManager) FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject {
	namespaceId, err := data.GetString("namespace_id")
	if err != nil {
		panic(fmt.Sprintf("get namespace_id from data %s error: %v", data, err))
	}
	return jsonutils.Marshal(map[string]string{
		"namespace_id": namespaceId,
	})
}

func (m *SNamespaceResourceBaseManager) FilterByUniqValues(q *sqlchemy.SQuery, values jsonutils.JSONObject) *sqlchemy.SQuery {
	namespaceId, _ := values.GetString("namespace_id")
	if len(namespaceId) > 0 {
		q = q.Equals("namespace_id", namespaceId)
	}
	return q
}

func (m *SNamespaceResourceBaseManager) GetByIdOrName(userCred mcclient.IIdentityProvider, clusterId, namespaceId string, resId string) (IClusterModel, error) {
	return FetchClusterResourceByIdOrName(m, userCred, clusterId, namespaceId, resId)
}

func (m *SNamespaceResourceBaseManager) GetByName(userCred mcclient.IIdentityProvider, clusterId, namespaceId string, resId string) (IClusterModel, error) {
	return FetchClusterResourceByName(m, userCred, clusterId, namespaceId, resId)
}

func (m SNamespaceResourceBaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.NamespaceResourceCreateInput) (*api.NamespaceResourceCreateInput, error) {
	cData, err := m.SClusterResourceBaseManager.ValidateCreateData(ctx, userCred, ownerCred, query, &data.ClusterResourceCreateInput)
	if err != nil {
		return nil, err
	}
	data.ClusterResourceCreateInput = *cData

	if data.NamespaceId == "" {
		return nil, httperrors.NewNotEmptyError("namespace is empty")
	}
	nsObj, err := GetNamespaceManager().GetByIdOrName(userCred, data.ClusterId, data.NamespaceId)
	if err != nil {
		return nil, NewCheckIdOrNameError("namespace_id", data.NamespaceId, err)
	}
	data.NamespaceId = nsObj.GetId()
	data.Namespace = nsObj.GetName()
	remoteNs, err := nsObj.GetRemoteObject()
	if err != nil {
		return nil, errors.Wrapf(err, "get remote namespace %s", remoteNs.(metav1.Object).GetName())
	}
	return data, nil
}

func (res *SNamespaceResourceBase) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if err := res.SClusterResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data); err != nil {
		return err
	}
	input := new(api.NamespaceResourceCreateInput)
	if err := data.Unmarshal(input); err != nil {
		return errors.Wrap(err, "namespace resource unmarshal data")
	}
	res.NamespaceId = input.NamespaceId
	return nil
}

func (res *SNamespaceResourceBase) GetNamespaceId() string {
	return res.NamespaceId
}

func (res *SNamespaceResourceBase) GetNamespace() (*SNamespace, error) {
	obj, err := GetNamespaceManager().FetchById(res.NamespaceId)
	if err != nil {
		return nil, errors.Wrapf(err, "fetch namespace %s", res.NamespaceId)
	}
	return obj.(*SNamespace), nil
}

func (res *SNamespaceResourceBase) GetNamespaceName() (string, error) {
	ns, err := res.GetNamespace()
	if err != nil {
		return "", err
	}
	return ns.GetName(), nil
}

type INamespaceModel interface {
	IClusterModel

	GetNamespaceId() string
	GetNamespace() (*SNamespace, error)
	SetNamespace(userCred mcclient.TokenCredential, ns *SNamespace)
}

func (m SNamespaceResourceBaseManager) IsNamespaceScope() bool {
	return true
}

func (m *SNamespaceResourceBaseManager) IsRemoteObjectLocalExist(userCred mcclient.TokenCredential, cluster *SCluster, obj interface{}) (IClusterModel, bool, error) {
	metaObj := obj.(metav1.Object)
	objName := metaObj.GetName()
	objNs := metaObj.GetNamespace()
	localNs, err := GetNamespaceManager().GetByName(userCred, cluster.GetId(), objNs)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, errors.Wrapf(err, "get cluster %s namespace %s for %s", cluster.GetId(), objNs, m.Keyword())
	}
	if localObj, _ := m.GetByName(userCred, cluster.GetId(), localNs.GetId(), objName); localObj != nil {
		return localObj, true, nil
	}
	return nil, false, nil
}

func (res *SNamespaceResourceBaseManager) NewFromRemoteObject(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	cluster *SCluster,
	remoteObj interface{}) (IClusterModel, error) {
	clsObj, err := res.SClusterResourceBaseManager.NewFromRemoteObject(ctx, userCred, cluster, remoteObj)
	if err != nil {
		return nil, errors.Wrap(err, "call cluster resource base NewFromRemoteObject")
	}
	localObj := clsObj.(INamespaceModel)
	ns := remoteObj.(metav1.Object).GetNamespace()
	localNs, err := GetNamespaceManager().GetByName(userCred, cluster.GetId(), ns)
	if err != nil {
		return nil, errors.Wrapf(err, "get local namespace by name %s when NewFromRemoteObject", ns)
	}
	localObj.SetNamespace(userCred, localNs.(*SNamespace))
	return localObj, nil
}

func (res *SNamespaceResourceBase) UpdateFromRemoteObject(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	extObj interface{},
) error {
	if err := res.SClusterResourceBase.UpdateFromRemoteObject(ctx, userCred, extObj); err != nil {
		return errors.Wrap(err, "SClusterResourceBase.UpdateFromRemoteObject")
	}
	cluster, err := res.GetCluster()
	if err != nil {
		return errors.Wrap(err, "SNamespaceResourceBase.GetCluster")
	}
	ns := extObj.(metav1.Object).GetNamespace()
	localNs, err := GetNamespaceManager().GetByName(userCred, cluster.GetId(), ns)
	if err != nil {
		return errors.Wrapf(err, "get local namespace by name %s when UpdateFromRemoteObject", ns)
	}
	res.SetNamespace(userCred, localNs.(*SNamespace))
	return nil
}

func (m *SNamespaceResourceBaseManager) GetGCQuery() *sqlchemy.SQuery {
	q := m.SClusterResourceBaseManager.GetGCQuery()
	nsIds := GetNamespaceManager().Query("id").SubQuery()
	q = q.Filter(sqlchemy.OR(
		sqlchemy.NotIn(q.Field("namespace_id"), nsIds),
	))
	return q
}

func (m *SNamespaceResourceBaseManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.NamespaceResourceListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SClusterResourceBaseManager.ListItemFilter(ctx, q, userCred, &input.ClusterResourceListInput)
	if err != nil {
		return nil, errors.Wrapf(err, "SClusterResourceBaseManager.ListItemFilter with input: %s", jsonutils.Marshal(input.ClusterResourceListInput))
	}
	log.Infof("=====namespace list input: %s", jsonutils.Marshal(input))
	if input.Namespace != "" {
		ns, err := GetNamespaceManager().GetByIdOrName(userCred, input.ClusterId, input.Namespace)
		if err != nil {
			return nil, errors.Wrapf(err, "Not found namespace %s by cluster %s", input.Namespace, input.Cluster)
		}
		log.Infof("=====get ns %s of id: %s", ns.GetName(), ns.GetId())
		q = q.Equals("namespace_id", ns.GetId())
	}
	return q, nil
}

func (res SNamespaceResourceBase) GetObjectMeta() (api.ObjectMeta, error) {
	return NewObjectMeta(&res)
}

func (res *SNamespaceResourceBase) SetNamespace(userCred mcclient.TokenCredential, ns *SNamespace) {
	res.NamespaceId = ns.GetId()
}

func (m *SNamespaceResourceBaseManager) FilterByHiddenSystemAttributes(q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	return m.SClusterResourceBaseManager.FilterBySystemAttributes(q, userCred, query, scope)
	//input := new(api.NamespaceResourceListInput)
	//if err := query.Unmarshal(input); err != nil {
	//log.Errorf("unmarshal namespace resource list input error: %v", err)
	//}
	//isSystem := false
	//if input.System != nil {
	//isSystem = *input.System
	//}
	//nsQ := NamespaceManager.Query("id")
	//nsSq := nsQ.Equals("name", userCred.GetProjectId())
	//if !isSystem {
	//q = q.Filter(
	//sqlchemy.OR(
	//sqlchemy.In(q.Field("namespace_id"), nsSq.SubQuery()),
	//),
	//)
	//}
	//return q
}

func (m *SNamespaceResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []interface{} {
	return m.SClusterResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
}

func (obj *SNamespaceResourceBase) GetDetails(ctx context.Context, cli *client.ClusterManager, base interface{}, k8sObj runtime.Object, isList bool) interface{} {
	out := api.NamespaceResourceDetail{
		ClusterResourceDetail: obj.SClusterResourceBase.GetDetails(ctx, cli, base, k8sObj, isList).(api.ClusterResourceDetail),
	}
	ns, err := obj.GetNamespace()
	if err != nil {
		log.Errorf("Get resource %s namespace error: %v", obj.GetName(), err)
	} else {
		out.Namespace = ns.GetName()
		out.NamespaceId = ns.GetId()

		remoteNs, _ := cli.GetClientset().CoreV1().Namespaces().Get(ctx, ns.GetName(), metav1.GetOptions{})
		if remoteNs != nil {
			out.NamespaceLabels = remoteNs.GetLabels()
		}
	}
	return out
}

func (res *SNamespaceResourceBase) GetRemoteObject() (interface{}, error) {
	cli, err := res.GetClusterClient()
	if err != nil {
		return nil, err
	}
	ns, err := res.GetNamespace()
	if err != nil {
		return nil, errors.Wrap(err, "get namespace")
	}
	resInfo := res.GetClusterModelManager().GetK8sResourceInfo()
	k8sCli := cli.GetHandler()
	return k8sCli.Get(resInfo.ResourceName, ns.GetName(), res.GetName())
}

func (res *SNamespaceResourceBase) DeleteRemoteObject() error {
	resInfo := res.GetClusterModelManager().GetK8sResourceInfo()
	cli, err := res.GetClusterClient()
	if err != nil {
		return err
	}
	ns, err := res.GetNamespace()
	if err != nil {
		return errors.Wrap(err, "get namespace")
	}
	if err := cli.GetHandler().Delete(resInfo.ResourceName, ns.GetName(), res.GetName(), &metav1.DeleteOptions{}); err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

func (res *SNamespaceResourceBase) AllowGetDetailsRawdata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	//return res.AllowGetDetails()
	// TODO: use rbac to check
	return true
}

func (res *SNamespaceResourceBase) GetDetailsRawdata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	k8sObj, err := GetK8sObject(res)
	if err != nil {
		return nil, err
	}
	return K8SObjectToJSONObject(k8sObj), nil
}

func (res *SNamespaceResourceBase) AllowUpdateRawdata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return true
}

func (res *SNamespaceResourceBase) UpdateRawdata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	k8sObj, err := UpdateK8sObject(res, data)
	if err != nil {
		return nil, err
	}
	return K8SObjectToJSONObject(k8sObj), nil
}
