package model

import (
	"strings"

	batch "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
)

type SK8sClusterResourceBase struct {
	SK8sModelBase

	Cluster string `json:"cluster"`
}

type SK8sClusterResourceBaseManager struct {
	SK8sModelBaseManager
}

func NewK8SClusterResourceBaseManager(dt interface{}, keyword, keywordPlural string) SK8sClusterResourceBaseManager {
	m := SK8sClusterResourceBaseManager{
		NewK8sModelBaseManager(dt, keyword, keywordPlural),
	}
	m.RegisterOrderFields(
		OrderFieldCreationTimestamp{},
		OrderFieldName())
	return m
}

func (_ SK8sClusterResourceBaseManager) GetOwnerModel(userCred mcclient.TokenCredential, manager IModelManager, cluster ICluster, namespace string, nameOrId string) (IOwnerModel, error) {
	return NewK8SModelObjectByName(manager.(IK8sModelManager), cluster, namespace, nameOrId)
}

func (m *SK8sClusterResourceBaseManager) ListItemFilter(ctx *RequestContext, q IQuery, query api.ListInputK8SClusterBase) (IQuery, error) {
	if query.Name != "" {
		q.AddFilter(func(obj IK8sModel) (bool, error) {
			return obj.GetName() == query.Name || strings.Contains(obj.GetName(), query.Name), nil
		})
	}
	return m.SK8sModelBaseManager.ListItemFilter(ctx, q, query.ListInputK8SBase)
}

func (m *SK8sClusterResourceBaseManager) ValidateCreateData(
	ctx *RequestContext,
	_ *jsonutils.JSONDict,
	input *api.K8sClusterResourceCreateInput) (*api.K8sClusterResourceCreateInput, error) {
	if input.Cluster == "" {
		return nil, httperrors.NewNotEmptyError("cluster is empty")
	}
	return input, nil
}

func (m *SK8sClusterResourceBase) ValidateUpdateData(
	_ *RequestContext, query, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewNotAcceptableError("%s not support update", m.Keyword())
}

func (m *SK8sClusterResourceBase) ValidateDeleteCondition(
	_ *RequestContext, _, _ *jsonutils.JSONDict) error {
	return nil
}

func (m *SK8sClusterResourceBase) CustomizeDelete(
	ctx *RequestContext, _, _ *jsonutils.JSONDict) error {
	return nil
}

func (m *SK8sClusterResourceBase) GetName() string {
	meta, _ := m.GetObjectMeta()
	return meta.GetName()
}

type SK8sNamespaceResourceBase struct {
	SK8sClusterResourceBase
}

type SK8sNamespaceResourceBaseManager struct {
	SK8sClusterResourceBaseManager
}

func NewK8sNamespaceResourceBaseManager(dt interface{}, keyword string, keywordPlural string) SK8sNamespaceResourceBaseManager {
	man := SK8sNamespaceResourceBaseManager{NewK8SClusterResourceBaseManager(dt, keyword, keywordPlural)}
	man.RegisterOrderFields(
		OrderFieldNamespace(),
		OrderFieldStatus())
	return man
}

func (m *SK8sNamespaceResourceBaseManager) ListItemFilter(ctx *RequestContext, q IQuery, query api.ListInputK8SNamespaceBase) (IQuery, error) {
	if query.Namespace != "" {
		q.Namespace(query.Namespace)
		/*q.AddFilter(func(obj metav1.Object) bool {
			return obj.GetNamespace() != query.Namespace
		})*/
	}
	return m.SK8sClusterResourceBaseManager.ListItemFilter(ctx, q, query.ListInputK8SClusterBase)
}

func (m SK8sNamespaceResourceBaseManager) ValidateCreateData(
	ctx *RequestContext, query *jsonutils.JSONDict,
	input *api.K8sNamespaceResourceCreateInput) (*api.K8sNamespaceResourceCreateInput, error) {
	cInput, err := m.SK8sClusterResourceBaseManager.ValidateCreateData(ctx, query, &input.K8sClusterResourceCreateInput)
	if err != nil {
		return nil, err
	}
	// TODO: check namespace resource exists
	input.K8sClusterResourceCreateInput = *cInput
	return input, nil
}

func (m SK8sNamespaceResourceBase) GetNamespace() string {
	return m.GetMetaObject().GetNamespace()
}

type SK8sOwnedResourceBaseManager struct{}

type IK8sOwnedResource interface {
	IsOwnedBy(ownerModel IOwnerModel) (bool, error)
}

func (m SK8sOwnedResourceBaseManager) ListItemFilter(ctx *RequestContext, q IQuery, query api.ListInputOwner) (IQuery, error) {
	if !query.ShouldDo() {
		return q, nil
	}
	q.AddFilter(m.ListOwnerFilter(ctx.UserCred(), query))
	return q, nil
}

func (m SK8sOwnedResourceBaseManager) ListOwnerFilter(userCred mcclient.TokenCredential, query api.ListInputOwner) QueryFilter {
	return func(obj IK8sModel) (bool, error) {
		man := GetK8sModelManagerByKind(query.OwnerKind)
		if man == nil {
			return false, httperrors.NewNotFoundError("Not found owner_kind %s", query.OwnerKind)
		}
		ownerModel, err := man.GetOwnerModel(userCred, man, obj.GetCluster(), obj.GetNamespace(), query.OwnerName)
		if err != nil {
			return false, err
		}
		return obj.(IK8sOwnedResource).IsOwnedBy(ownerModel)
	}
}

func IsEventOwner(model IOwnerModel, event *v1.Event) (bool, error) {
	metaObj, err := model.GetObjectMeta()
	if err != nil {
		return false, err
	}
	return event.InvolvedObject.UID == metaObj.GetUID(), nil
}

func IsJobOwner(model IK8sModel, job *batch.Job) (bool, error) {
	metaObj, err := model.GetObjectMeta()
	if err != nil {
		return false, err
	}
	for _, i := range job.OwnerReferences {
		if i.UID == metaObj.GetUID() {
			return true, nil
		}
	}
	return false, nil
}
