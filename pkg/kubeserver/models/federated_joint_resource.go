package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
)

var (
	globalFedJointClusterManagers map[string]IFedJointClusterManager
)

func RegisterFedJointClusterManager(masterMan IFedModelManager, jointMan IFedJointClusterManager) {
	if globalFedJointClusterManagers == nil {
		globalFedJointClusterManagers = make(map[string]IFedJointClusterManager)
	}
	globalFedJointClusterManagers[masterMan.Keyword()] = jointMan
}

func GetFedJointClusterManager(keyword string) IFedJointClusterManager {
	return globalFedJointClusterManagers[keyword]
}

// GetFedJointNamespaceScopeManager return federated namespace scope manager,
// e.g:
// - GetFedRoleManager()
// - GetFedRoleBindingManager()
func GetFedJointNamespaceScopeManager() []IFedNamespaceJointClusterManager {
	ret := make([]IFedNamespaceJointClusterManager, 0)
	for _, m := range globalFedJointClusterManagers {
		if m.GetResourceManager().IsNamespaceScope() {
			ret = append(ret, m)
		}
	}
	return ret
}

func GetFedManagers() []IFedModelManager {
	ret := make([]IFedModelManager, 0)
	for _, m := range globalFedJointClusterManagers {
		ret = append(ret, m.GetFedManager())
	}
	return ret
}

type IFedJointModel interface {
	db.IJointModel
}

type IFedJointManager interface {
	db.IJointModelManager
}

type IFedJointClusterManager interface {
	IFedJointManager
	GetFedManager() IFedModelManager
	GetResourceManager() IClusterModelManager
	ClusterQuery(clusterId string) *sqlchemy.SQuery
}

type IFedJointClusterModel interface {
	IFedJointModel

	// GetClusterId() get object cluster_id
	GetClusterId() string
	// GetResourceId get object resource_id
	GetResourceId() string
	// GetFedResourceId get object federatedresource_id
	GetFedResourceId() string
	GetManager() IFedJointClusterManager
	GetCluster() (*SCluster, error)
	GetResourceManager() IClusterModelManager
	SetResource(resObj IClusterModel) error
	GetResourceCreateData(ctx context.Context, userCred mcclient.TokenCredential, fedObj IFedModel, baseInput api.NamespaceResourceCreateInput) (jsonutils.JSONObject, error)
	GetResourceUpdateData(ctx context.Context, userCred mcclient.TokenCredential, fedObj IFedModel, resObj IClusterModel, baseInput api.NamespaceResourceUpdateInput) (jsonutils.JSONObject, error)
	GetDetails(base api.FedJointClusterResourceDetails, isList bool) interface{}
}

// +onecloud:swagger-gen-ignore
type SFedJointResourceBaseManager struct {
	db.SJointResourceBaseManager
}

type SFedJointResourceBase struct {
	db.SJointResourceBase
}

func NewFedJointResourceBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string, master IFedModelManager, slave db.IStandaloneModelManager) SFedJointResourceBaseManager {
	return SFedJointResourceBaseManager{
		SJointResourceBaseManager: db.NewJointResourceBaseManager(dt, tableName, keyword, keywordPlural, master.(db.IStandaloneModelManager), slave),
	}
}

func NewFedJointClusterManager(
	dt interface{}, tableName string,
	keyword string, keywordPlural string,
	master IFedModelManager,
	resourceMan IClusterModelManager,
) SFedJointClusterManager {
	base := NewFedJointResourceBaseManager(dt, tableName, keyword, keywordPlural, master, GetClusterManager())
	man := SFedJointClusterManager{
		SFedJointResourceBaseManager: base,
		resourceManager:              resourceMan,
	}
	return man
}

func NewFedJointManager(factory func() db.IJointModelManager) db.IJointModelManager {
	man := factory()
	man.SetVirtualObject(man)
	return man
}

type SFedJointClusterManager struct {
	SFedJointResourceBaseManager
	resourceManager IClusterModelManager
}

type SFedJointCluster struct {
	SFedJointResourceBase

	FederatedresourceId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
	ClusterId           string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
	// NamespaceId should be calculated by cluster and federated resource
	ResourceId string `width:"36" charset:"ascii" list:"user" index:"true"`
}

func (m SFedJointClusterManager) GetFedManager() IFedModelManager {
	return m.GetMasterManager().(IFedModelManager)
}

func (m SFedJointClusterManager) GetResourceManager() IClusterModelManager {
	return m.resourceManager
}

func (m SFedJointClusterManager) GetMasterFieldName() string {
	return "federatedresource_id"
}

func (m SFedJointClusterManager) GetSlaveFieldName() string {
	return "cluster_id"
}

func (m *SFedJointClusterManager) ClusterQuery(clsId string) *sqlchemy.SQuery {
	return m.Query().Equals("cluster_id", clsId)
}

func GetFederatedJointClusterModel(man IFedJointClusterManager, masterId string, clusterId string) (IFedJointClusterModel, error) {
	q := man.ClusterQuery(clusterId).Equals(man.GetMasterFieldName(), masterId)
	obj, err := db.NewModelObject(man)
	if err != nil {
		return nil, errors.Wrapf(err, "NewModelObject %s", man.Keyword())
	}
	if err := q.First(obj); err != nil {
		return nil, err
	}
	return obj.(IFedJointClusterModel), nil
}

func (obj *SFedJointCluster) GetManager() IFedJointClusterManager {
	return obj.GetJointModelManager().(IFedJointClusterManager)
}

func (obj *SFedJointCluster) GetResourceManager() IClusterModelManager {
	return obj.GetManager().GetResourceManager()
}

func (obj *SFedJointCluster) GetFedResourceManager() IFedModelManager {
	return obj.GetManager().GetFedManager()
}

func (obj *SFedJointCluster) GetClusterId() string {
	return obj.ClusterId
}

func (obj *SFedJointCluster) GetFedResourceId() string {
	return obj.FederatedresourceId
}

func (obj *SFedJointCluster) GetResourceId() string {
	return obj.ResourceId
}

func (obj *SFedJointCluster) SetResource(resObj IClusterModel) error {
	_, err := db.Update(obj, func() error {
		obj.ResourceId = resObj.GetId()
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "set resource_id")
	}
	return nil
}

func (m *SFedJointClusterManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.FedJointClusterListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SFedJointResourceBaseManager.ListItemFilter(ctx, q, userCred, input.JointResourceBaseListInput)
	if err != nil {
		return nil, err
	}
	if len(input.FederatedResourceId) > 0 {
		masterMan := m.GetMasterManager()
		fedObj, err := masterMan.FetchByIdOrName(userCred, input.FederatedResourceId)
		if err != nil {
			return nil, errors.Wrapf(err, "Get %s object", masterMan.Keyword())
		}
		q = q.Equals("federatedresource_id", fedObj.GetId())
	}
	if len(input.ClusterId) > 0 {
		clusterObj, err := GetClusterManager().FetchByIdOrName(userCred, input.ClusterId)
		if err != nil {
			return nil, errors.Wrap(err, "Get cluster")
		}
		q = q.Equals("cluster_id", clusterObj.GetId())
	}
	if len(input.ClusterName) > 0 {
		csq := GetClusterManager().Query("id").Contains("name", input.ClusterName).SubQuery()
		q = q.In("cluster_id", csq)
	}
	if len(input.ResourceId) > 0 {
		resObj, err := m.GetResourceManager().FetchByIdOrName(userCred, input.ResourceId)
		if err != nil {
			return nil, errors.Wrap(err, "Get resource")
		}
		q = q.Equals("resource_id", resObj.GetId())
	}
	if len(input.ResourceName) > 0 {
		rsq := m.GetResourceManager().Query("id").Contains("name", input.ResourceName).SubQuery()
		q = q.In("resource_id", rsq)
	}
	/*
	 * if len(input.NamespaceId) > 0 {
	 *     nsObj, err := GetNamespaceManager().FetchByIdOrName(userCred, input.NamespaceId)
	 *     if err != nil {
	 *         return nil, errors.Wrap(err, "Get namespace")
	 *     }
	 *     q = q.Equals("namespace_id", nsObj.GetId())
	 * }
	 */
	return q, nil
}

func (m *SFedJointClusterManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []interface{}, fields stringutils2.SSortedStrings, isList bool) []interface{} {
	baseGet := func(obj interface{}) interface{} {
		jRows := m.SJointResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, []interface{}{obj}, fields, isList)
		return jRows[0]
	}
	ret := make([]interface{}, len(objs))
	fedApi := GetFedResAPI()
	for idx := range objs {
		obj := objs[idx].(IFedJointClusterModel)
		baseDetail := baseGet(obj)
		out := fedApi.JointResAPI().GetDetails(ctx, obj, userCred, baseDetail.(apis.JointResourceBaseDetails), isList)
		ret[idx] = out
	}
	return ret
}

func (obj *SFedJointCluster) GetCluster() (*SCluster, error) {
	return GetClusterManager().GetCluster(obj.ClusterId)
}

func (obj *SFedJointCluster) GetDetails(base api.FedJointClusterResourceDetails, isList bool) interface{} {
	return base
}
