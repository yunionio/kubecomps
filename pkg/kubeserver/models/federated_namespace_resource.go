package models

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
)

type IFedNamespaceModel interface {
	IFedModel

	GetFedNamespace() (*SFedNamespace, error)
}

// +onecloud:swagger-gen-ignore
type SFedNamespaceResourceManager struct {
	SFedResourceBaseManager
}

type SFedNamespaceResource struct {
	SFedResourceBase

	FederatednamespaceId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
}

func NewFedNamespaceResourceManager(
	dt interface{},
	tableName string,
	keyword string,
	keywordPlural string,
) SFedNamespaceResourceManager {
	return SFedNamespaceResourceManager{
		SFedResourceBaseManager: NewFedResourceBaseManager(dt, tableName, keyword, keywordPlural),
	}
}

func (m *SFedNamespaceResourceManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.FederatedNamespaceResourceCreateInput) (*api.FederatedNamespaceResourceCreateInput, error) {
	rInput, err := m.SFedResourceBaseManager.ValidateCreateData(ctx, userCred, ownerCred, query, &input.FederatedResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.FederatedResourceCreateInput = *rInput
	fedNsId := input.FederatednamespaceId
	if fedNsId == "" {
		return nil, httperrors.NewNotEmptyError("federatednamespace_id is empty")
	}
	nsObj, err := GetFedNamespaceManager().GetFedNamespaceByIdOrName(userCred, fedNsId)
	if err != nil {
		return nil, err
	}
	input.FederatednamespaceId = nsObj.GetId()
	input.Federatednamespace = nsObj.GetName()
	return input, nil
}

func (m *SFedNamespaceResourceManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.FederatedNamespaceResourceListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SFedResourceBaseManager.ListItemFilter(ctx, q, userCred, &input.FederatedResourceListInput)
	if err != nil {
		return nil, err
	}
	if input.FederatednamespaceId != "" {
		ns, err := GetFedNamespaceManager().GetFedNamespaceByIdOrName(userCred, input.FederatednamespaceId)
		if err != nil {
			return nil, err
		}
		q = q.Equals("federatednamespace_id", ns.GetId())
	}
	return q, nil
}

func (obj *SFedNamespaceResource) GetFedNamespace() (*SFedNamespace, error) {
	return GetFedNamespaceManager().GetFedNamespace(obj.FederatednamespaceId)
}

func (obj *SFedNamespaceResource) ValidateAttachCluster(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	data, err := obj.SFedResourceBase.ValidateAttachCluster(ctx, userCred, data)
	if err != nil {
		return nil, err
	}
	// fedNs, err := obj.GetFedNamespace()
	_, err = obj.GetFedNamespace()
	if err != nil {
		return nil, errors.Wrap(err, "get federated namespace")
	}
	// clusterId, err := data.GetString("cluster_id")
	_, err = data.GetString("cluster_id")
	if err != nil {
		return nil, errors.Wrap(err, "get cluster_id")
	}
	/*
	 * nsObj, err := GetNamespaceManager().GetByIdOrName(userCred, clusterId, fedNs.GetName())
	 * if err != nil {
	 *     return nil, errors.Wrapf(err, "get cluster %s namespace %s", clusterId, fedNs.GetName())
	 * }
	 * data.(*jsonutils.JSONDict).Set("namespace_id", jsonutils.NewString(nsObj.GetId()))
	 * data.(*jsonutils.JSONDict).Set("namespace_name", jsonutils.NewString(nsObj.GetName()))
	 */
	return data, nil
}

func (obj *SFedNamespaceResource) PerformAttachCluster(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	elemObj, err := obj.GetElemModel()
	if err != nil {
		return nil, err
	}
	if err := GetFedResAPI().NamespaceScope().StartAttachClusterTask(elemObj.(IFedNamespaceModel), ctx, userCred, data.(*jsonutils.JSONDict), ""); err != nil {
		return nil, errors.Wrap(err, "StartAttachClusterTask")
	}
	// hack sleep 1 seconds to wait joint model created
	// TODO: fix this
	time.Sleep(1 * time.Second)
	return nil, nil
}

func (obj *SFedNamespaceResource) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.FedNamespaceResourceUpdateInput) (*api.FedNamespaceResourceUpdateInput, error) {
	bInput, err := obj.SFedResourceBase.ValidateUpdateData(ctx, userCred, query, &input.FedResourceUpdateInput)
	if err != nil {
		return nil, err
	}
	input.FedResourceUpdateInput = *bInput
	return input, nil
}

func (obj *SFedNamespaceResource) GetK8sObjectMeta() metav1.ObjectMeta {
	objMeta := obj.SFedResourceBase.GetK8sObjectMeta()
	fedNs, err := obj.GetFedNamespace()
	if err != nil {
		log.Errorf("GetK8sObjectMeta federated namespace error: %v", err)
	}
	if fedNs != nil {
		objMeta.Namespace = fedNs.GetName()
	}
	return objMeta
}

func (obj *SFedNamespaceResource) GetDetails(base interface{}, isList bool) interface{} {
	out := api.FederatedNamespaceResourceDetails{
		FederatedResourceDetails: obj.SFedResourceBase.GetDetails(base, isList).(api.FederatedResourceDetails),
	}
	fedNs, err := obj.GetFedNamespace()
	if err != nil {
		log.Errorf("get federatednamespace error: %v", err)
	} else {
		out.Federatednamespace = fedNs.GetName()
	}
	return out
}
