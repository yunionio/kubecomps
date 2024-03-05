package models

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	"k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/apis/core/validation"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/utils/k8serrors"
)

var (
	fedNamespaceManager *SFedNamespaceManager
	_                   IFedModelManager = new(SFedNamespaceManager)
	_                   IFedModel        = new(SFedNamespace)
)

func init() {
	GetFedNamespaceManager()
}

func GetFedNamespaceManager() *SFedNamespaceManager {
	if fedNamespaceManager == nil {
		fedNamespaceManager = newModelManager(func() db.IModelManager {
			return &SFedNamespaceManager{
				SFedResourceBaseManager: NewFedResourceBaseManager(
					SFedNamespace{},
					"federatednamespaces_tbl",
					"federatednamespace",
					"federatednamespaces",
				),
			}
		}).(*SFedNamespaceManager)
	}
	return fedNamespaceManager
}

// +onecloud:swagger-gen-model-singular=federatednamespace
// +onecloud:swagger-gen-model-plural=federatednamespaces
type SFedNamespaceManager struct {
	SFedResourceBaseManager
}

type SFedNamespace struct {
	SFedResourceBase
	Spec *api.FederatedNamespaceSpec `list:"user" update:"user" create:"required"`
}

func (m *SFedNamespaceManager) GetFedNamespace(id string) (*SFedNamespace, error) {
	obj, err := m.FetchById(id)
	if err != nil {
		return nil, err
	}
	return obj.(*SFedNamespace), nil
}

func (m *SFedNamespaceManager) GetFedNamespaceByIdOrName(ctx context.Context, userCred mcclient.IIdentityProvider, id string) (*SFedNamespace, error) {
	obj, err := m.FetchByIdOrName(ctx, userCred, id)
	if err != nil {
		return nil, err
	}
	return obj.(*SFedNamespace), nil
}

func (m *SFedNamespaceManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.FederatedNamespaceCreateInput) (*api.FederatedNamespaceCreateInput, error) {
	rInput, err := m.SFedResourceBaseManager.ValidateCreateData(ctx, userCred, ownerCred, query, &input.FederatedResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.FederatedResourceCreateInput = *rInput
	nsObj := input.ToNamespace()
	out, err := m.K8sConvertToInternalObject(nsObj)
	if err != nil {
		return nil, err
	}
	if err := validation.ValidateNamespace(out.(*core.Namespace)).ToAggregate(); err != nil {
		return nil, httperrors.NewInputParameterError("%s", err)
	}
	return input, nil
}

func (obj *SFedNamespace) GetDetails(base interface{}, isList bool) interface{} {
	out := api.FederatedNamespaceDetails{
		FederatedResourceDetails: obj.SFedResourceBase.GetDetails(base, isList).(api.FederatedResourceDetails),
	}
	return out
}

// ValidateDeleteCondition check steps:
// 1. no releated clusters attached
// 2. no releated federated namespace scope resource
func (obj *SFedNamespace) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if err := obj.SFedResourceBase.ValidateDeleteCondition(ctx, info); err != nil {
		return err
	}
	fedNsMans := GetFedJointNamespaceScopeManager()
	fedApi := GetFedResAPI().JointResAPI().NamespaceScope()
	for _, m := range fedNsMans {
		objs, err := fedApi.FetchModelsByFednamespace(m, obj.GetId())
		if err != nil {
			return errors.Wrapf(err, "fetch %s models by fednamespace", m.Keyword())
		}
		if len(objs) > 0 {
			return httperrors.NewNotEmptyError("federatednamespace %s has %d related %s attached", obj.GetName(), len(objs), m.Keyword())
		}
	}
	return nil
}

func (obj *SFedNamespace) PerformAttachCluster(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *api.FederatedNamespaceAttachClusterInput) (*api.FederatedNamespaceAttachClusterInput, error) {
	_, err := obj.SFedResourceBase.PerformAttachCluster(ctx, userCred, query, data.JSON(data))
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (obj *SFedNamespace) PerformDetachCluster(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *api.FederatedNamespaceDetachClusterInput) (*api.FederatedNamespaceDetachClusterInput, error) {
	_, err := obj.SFedResourceBase.PerformDetachCluster(ctx, userCred, query, data.JSON(data))
	return nil, err
}

func (m *SFedNamespaceManager) K8sConvertToInternalObject(in *corev1.Namespace) (interface{}, error) {
	out := new(core.Namespace)
	if err := legacyscheme.Scheme.Convert(in, out, nil); err != nil {
		return nil, errors.Wrap(k8serrors.NewGeneralError(err), "convert to internal namespace object")
	}
	return out, nil
}

func (obj *SFedNamespace) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.FedNamespaceUpdateInput) (*api.FedNamespaceUpdateInput, error) {
	rInput, err := obj.SFedResourceBase.ValidateUpdateData(ctx, userCred, query, &input.FedResourceUpdateInput)
	if err != nil {
		return nil, errors.Wrap(err, "SFedResourceBase.ValidateUpdateData")
	}
	input.FedResourceUpdateInput = *rInput
	nsObj := input.ToNamespace(obj.GetK8sObjectMeta())
	out, err := GetFedNamespaceManager().K8sConvertToInternalObject(nsObj)
	if err != nil {
		return nil, err
	}
	if err := validation.ValidateNamespace(out.(*core.Namespace)).ToAggregate(); err != nil {
		return nil, httperrors.NewInputParameterError("%s", err)
	}
	return input, nil
}
