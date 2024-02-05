package models

import (
	"context"
	"database/sql"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
)

type IFedModelManager interface {
	db.IModelManager

	GetJointModelManager() IFedJointClusterManager
	SetJointModelManager(man IFedJointClusterManager)
	PurgeAllByCluster(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster) error
}

type IFedModel interface {
	db.IStatusDomainLevelModel

	GetManager() IFedModelManager
	GetDetails(baseDetails interface{}, isList bool) interface{}
	ValidateJointCluster(userCred mcclient.TokenCredential, data jsonutils.JSONObject) (IFedJointClusterModel, jsonutils.JSONObject, error)
	GetJointModelManager() IFedJointClusterManager
	ValidateAttachCluster(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) (jsonutils.JSONObject, error)
	ValidateDetachCluster(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) (jsonutils.JSONObject, error)
	SetStatus(ctx context.Context, userCred mcclient.TokenCredential, status string, reason string) error
	LogPrefix() string
}

// +onecloud:swagger-gen-ignore
type SFedResourceBaseManager struct {
	db.SStatusDomainLevelResourceBaseManager
	jointManager IFedJointClusterManager
}

type SFedResourceBase struct {
	db.SStatusDomainLevelResourceBase
}

func NewFedResourceBaseManager(
	dt interface{},
	tableName string,
	keyword string,
	keywordPlural string,
) SFedResourceBaseManager {
	return SFedResourceBaseManager{
		SStatusDomainLevelResourceBaseManager: db.NewStatusDomainLevelResourceBaseManager(
			dt, tableName, keyword, keywordPlural),
	}
}

func (m *SFedResourceBaseManager) SetJointModelManager(man IFedJointClusterManager) {
	m.jointManager = man
}

func (m *SFedResourceBaseManager) GetJointModelManager() IFedJointClusterManager {
	return m.jointManager
}

func (m *SFedResourceBase) GetJointModelManager() IFedJointClusterManager {
	return m.GetManager().GetJointModelManager()
}

func (obj *SFedResourceBase) LogPrefix() string {
	return fmt.Sprintf("Federated %s %s(%s)", obj.Keyword(), obj.GetName(), obj.GetId())
}

func (m *SFedResourceBaseManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.FederatedResourceListInput) (*sqlchemy.SQuery, error) {
	return m.SStatusDomainLevelResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StatusDomainLevelResourceListInput)
}

func (m *SFedResourceBaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.FederatedResourceCreateInput) (*api.FederatedResourceCreateInput, error) {
	dInput, err := m.SStatusDomainLevelResourceBaseManager.ValidateCreateData(ctx, userCred, ownerCred, query, input.StatusDomainLevelResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.StatusDomainLevelResourceCreateInput = dInput
	return input, nil
}

func (m *SFedResourceBaseManager) FetchCustomizeColumns(
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
	ret := make([]interface{}, len(objs))
	for idx := range objs {
		obj := objs[idx].(IFedModel)
		baseDetail := baseGet(obj)
		out := obj.GetDetails(baseDetail, isList)
		ret[idx] = out
	}
	return ret
}

func (m *SFedResourceBaseManager) PurgeAllByCluster(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster) error {
	jm := m.GetJointModelManager()
	sq := jm.Query("federatedresource_id").Equals("cluster_id", cluster.GetId()).SubQuery()
	q := m.Query().In("id", sq)
	objs := make([]interface{}, 0)
	if err := db.FetchModelObjects(m, q, &objs); err != nil {
		return errors.Wrapf(err, "Fetch all %s objects when purge all", m.KeywordPlural())
	}
	fedApi := GetFedResAPI()
	for i := range objs {
		obj := objs[i]
		objPtr := GetObjectPtr(obj).(IFedModel)
		jObj, err := fedApi.GetJointModel(objPtr, cluster.GetId())
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				continue
			}
			return errors.Wrapf(err, "get joint model by cluster %s", cluster.GetId())
		}
		if jObj == nil {
			continue
		}
		if err := jObj.Delete(ctx, userCred); err != nil {
			return errors.Wrapf(err, "delete %s cluster %s joint model", objPtr.LogPrefix(), cluster.GetId())
		}
	}
	return nil
}

func (obj *SFedResourceBase) GetManager() IFedModelManager {
	return obj.GetModelManager().(IFedModelManager)
}

func (obj *SFedResourceBase) GetClustersQuery() *sqlchemy.SQuery {
	jointMan := obj.GetJointModelManager()
	return jointMan.Query().Equals(jointMan.GetMasterFieldName(), obj.GetId())
}

func (obj *SFedResourceBase) GetClustersCount() (int, error) {
	q := obj.GetClustersQuery()
	return q.CountWithError()
}

func (obj *SFedResourceBase) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	obj.SStatusDomainLevelResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	obj.SetStatus(ctx, userCred, api.FederatedResourceStatusActive, "post create")
}

func (obj *SFedResourceBase) GetDetails(base interface{}, isList bool) interface{} {
	out := api.FederatedResourceDetails{
		StatusDomainLevelResourceDetails: base.(apis.StatusDomainLevelResourceDetails),
	}
	clusterCount, err := obj.GetClustersCount()
	if err == nil {
		out.ClusterCount = &clusterCount
	} else {
		log.Errorf("Get %s cluster count error: %v", obj.LogPrefix(), err)
	}
	if isList {
		return out
	}
	// placement := api.FederatedPlacement{}
	return out
}

func (obj *SFedResourceBase) ValidateJointCluster(userCred mcclient.TokenCredential, data jsonutils.JSONObject) (IFedJointClusterModel, jsonutils.JSONObject, error) {
	jointMan := obj.GetJointModelManager()
	clusterId, _ := data.GetString("cluster_id")
	if clusterId == "" {
		return nil, data, httperrors.NewInputParameterError("cluster_id not provided")
	}
	cluster, err := GetClusterManager().GetClusterByIdOrName(userCred, clusterId)
	if err != nil {
		return nil, nil, err
	}
	clusterId = cluster.GetId()
	data.(*jsonutils.JSONDict).Set("cluster_id", jsonutils.NewString(clusterId))
	data.(*jsonutils.JSONDict).Set("cluster_name", jsonutils.NewString(cluster.GetName()))
	jointModel, err := GetFederatedJointClusterModel(jointMan, obj.GetId(), clusterId)
	return jointModel, data, err
}

func (obj *SFedResourceBase) ValidateAttachCluster(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	jointModel, data, err := obj.ValidateJointCluster(userCred, data)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return data, err
	}
	clusterId, _ := data.GetString("cluster_id")
	if jointModel != nil {
		return data, httperrors.NewInputParameterError("cluster %s has been attached", clusterId)
	}
	return data, nil
}

func (obj *SFedResourceBase) GetK8sObjectMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name: obj.Name,
	}
}

func (obj *SFedResourceBase) ValidateDetachCluster(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	jointModel, input, err := obj.ValidateJointCluster(userCred, data)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return input, err
	}
	clusterId, _ := data.GetString("cluster_id")
	if jointModel == nil {
		return input, httperrors.NewInputParameterError("cluster %s has not been attached", clusterId)
	}
	return input, nil
}

func (obj *SFedResourceBase) GetElemModel() (IFedModel, error) {
	m := obj.GetManager()
	elemObj, err := db.FetchById(m, obj.GetId())
	if err != nil {
		return nil, err
	}
	return elemObj.(IFedModel), nil
}

func (obj *SFedResourceBase) PerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	elemObj, err := obj.GetElemModel()
	if err != nil {
		return nil, err
	}
	return nil, GetFedResAPI().StartSyncTask(elemObj, ctx, userCred, data.(*jsonutils.JSONDict), "")
}

func (obj *SFedResourceBase) PerformSyncCluster(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *api.FederatedResourceJointClusterInput) (*api.FederatedResourceJointClusterInput, error) {
	elemObj, err := obj.GetElemModel()
	if err != nil {
		return nil, err
	}
	return nil, GetFedResAPI().PerformSyncCluster(elemObj, ctx, userCred, data.JSON(data))
}

func (obj *SFedResourceBase) PerformAttachCluster(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	elemObj, err := obj.GetElemModel()
	if err != nil {
		return nil, err
	}
	if _, err := GetFedResAPI().PerformAttachCluster(elemObj, ctx, userCred, data); err != nil {
		return nil, err
	}
	return nil, nil
}

func (obj *SFedResourceBase) PerformDetachCluster(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	elemObj, err := obj.GetElemModel()
	if err != nil {
		return nil, err
	}
	return nil, GetFedResAPI().PerformDetachCluster(elemObj, ctx, userCred, data)
}

func (m *SFedResourceBase) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.FedResourceUpdateInput) (*api.FedResourceUpdateInput, error) {
	bInput, err := m.SStatusDomainLevelResourceBase.ValidateUpdateData(ctx, userCred, query, input.StatusDomainLevelResourceBaseUpdateInput)
	if err != nil {
		return nil, err
	}
	input.StatusDomainLevelResourceBaseUpdateInput = bInput
	if input.Name != "" {
		return nil, httperrors.NewInputParameterError("Can not update name")
	}
	return input, nil
}

func (res *SFedResourceBase) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	res.SStatusDomainLevelResourceBase.PostUpdate(ctx, userCred, query, data)
	if err := GetFedResAPI().StartUpdateTask(res, ctx, userCred, data.(*jsonutils.JSONDict), ""); err != nil {
		log.Errorf("StartUpdateTask %s error: %v", res.LogPrefix(), err)
	}
}

func (obj *SFedResourceBase) ValidateDeleteCondition(ctx context.Context, _ jsonutils.JSONObject) error {
	clusters, err := GetFedResAPI().GetAttachedClusters(obj)
	if err != nil {
		return errors.Wrap(err, "get attached clusters")
	}
	clsName := make([]string, len(clusters))
	for i := range clusters {
		clsName[i] = clusters[i].GetName()
	}
	if len(clusters) != 0 {
		return httperrors.NewNotEmptyError("federated resource %s attached to cluster %v", obj.Keyword(), clsName)
	}
	return nil
}
