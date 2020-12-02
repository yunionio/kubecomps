package models

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/kubernetes/pkg/apis/rbac"
	"k8s.io/kubernetes/pkg/apis/rbac/validation"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
)

var (
	fedClusterRoleBindingManager *SFedClusterRoleBindingManager
	_                            IFedModelManager = new(SFedClusterRoleBindingManager)
	_                            IFedModel        = new(SFedClusterRoleBinding)
)

func init() {
	GetFedClusterRoleBindingManager()
}

func GetFedClusterRoleBindingManager() *SFedClusterRoleBindingManager {
	if fedClusterRoleBindingManager == nil {
		fedClusterRoleBindingManager = newModelManager(func() db.IModelManager {
			return &SFedClusterRoleBindingManager{
				SFedResourceBaseManager: NewFedResourceBaseManager(
					SFedClusterRoleBinding{},
					"federatedclusterrolebindings_tbl",
					"federatedclusterrolebinding",
					"federatedclusterrolebindings",
				),
			}
		}).(*SFedClusterRoleBindingManager)
	}
	return fedClusterRoleBindingManager
}

// +onecloud:swagger-gen-model-singular=federatedclusterrolebinding
// +onecloud:swagger-gen-model-plural=federatedclusterrolebindings
type SFedClusterRoleBindingManager struct {
	SFedResourceBaseManager
	// SRoleRefResourceBaseManager
}

type SFedClusterRoleBinding struct {
	SFedResourceBase
	Spec *api.FederatedClusterRoleBindingSpec `list:"user" update:"user" create:"required"`
	// SRoleRefResourceBase
}

func (m *SFedClusterRoleBindingManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.FederatedClusterRoleBindingCreateInput) (*api.FederatedClusterRoleBindingCreateInput, error) {
	fInput, err := m.SFedResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, &input.FederatedResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.FederatedResourceCreateInput = *fInput
	if err := ValidateFederatedRoleRef(ctx, userCred, input.Spec.Template.RoleRef); err != nil {
		return nil, err
	}
	crb := input.ToClusterRoleBinding()
	if err := GetClusterRoleBindingManager().ValidateClusterRoleBinding(crb); err != nil {
		return nil, err
	}
	return input, nil
}

func (obj *SFedClusterRoleBinding) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if err := obj.SFedResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data); err != nil {
		return err
	}
	return nil
}

func ValidateUpdateFedClusterRoleBindingObject(oldObj, newObj *rbacv1.ClusterRoleBinding) error {
	if err := ValidateUpdateK8sObject(oldObj, newObj, new(rbac.ClusterRoleBinding), new(rbac.ClusterRoleBinding), func(newObj, oldObj interface{}) field.ErrorList {
		crb := newObj.(*rbac.ClusterRoleBinding)
		oldCrb := oldObj.(*rbac.ClusterRoleBinding)
		allErrs := validation.ValidateClusterRoleBinding(crb)
		if oldCrb.RoleRef != crb.RoleRef {
			allErrs = append(allErrs, field.Invalid(field.NewPath("roleRef"), crb.RoleRef, "cannot change roleRef"))
		}
		return allErrs
	}); err != nil {
		return errors.Wrap(err, "ValidateUpdateFedClusterRoleBindingObject")
	}
	return nil
}

func (obj *SFedClusterRoleBinding) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.FedClusterRoleBindingUpdateInput) (*api.FedClusterRoleBindingUpdateInput, error) {
	bInput, err := obj.SFedResourceBase.ValidateUpdateData(ctx, userCred, query, &input.FedResourceUpdateInput)
	if err != nil {
		return nil, err
	}
	input.FedResourceUpdateInput = *bInput
	objMeta := obj.GetK8sObjectMeta()
	oldObj := obj.Spec.ToClusterRoleBinding(objMeta)
	newObj := input.ToClusterRoleBinding(objMeta)
	if err := ValidateUpdateFedClusterRoleBindingObject(oldObj, newObj); err != nil {
		return nil, err
	}
	return input, nil
}
