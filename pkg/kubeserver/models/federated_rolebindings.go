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
	fedRoleBindingManager *SFedRoleBindingManager
	_                     IFedModelManager = new(SFedRoleBindingManager)
	_                     IFedModel        = new(SFedRoleBinding)
)

func init() {
	GetFedRoleBindingManager()
}

func GetFedRoleBindingManager() *SFedRoleBindingManager {
	if fedRoleBindingManager == nil {
		fedRoleBindingManager = newModelManager(func() db.IModelManager {
			return &SFedRoleBindingManager{
				SFedNamespaceResourceManager: NewFedNamespaceResourceManager(
					SFedRoleBinding{},
					"federatedrolebindings_tbl",
					"federatedrolebinding",
					"federatedrolebindings",
				),
			}
		}).(*SFedRoleBindingManager)
	}
	return fedRoleBindingManager
}

// +onecloud:swagger-gen-model-singular=federatedrolebinding
// +onecloud:swagger-gen-model-plural=federatedrolebindings
type SFedRoleBindingManager struct {
	SFedNamespaceResourceManager
	// SRoleRefResourceBase
}

type SFedRoleBinding struct {
	SFedNamespaceResource
	Spec *api.FederatedRoleBindingSpec `list:"user" update:"user" create:"required"`
	// SRoleRefResourceBase
}

func (m *SFedRoleBindingManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.FederatedRoleBindingCreateInput) (*api.FederatedRoleBindingCreateInput, error) {
	nInput, err := m.SFedNamespaceResourceManager.ValidateCreateData(ctx, userCred, ownerCred, query, &input.FederatedNamespaceResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.FederatedNamespaceResourceCreateInput = *nInput
	if err := ValidateFederatedRoleRef(ctx, userCred, input.Spec.Template.RoleRef); err != nil {
		return nil, err
	}
	if err := GetRoleBindingManager().ValidateRoleBinding(input.ToRoleBinding(nInput.Federatednamespace)); err != nil {
		return nil, err
	}
	return input, nil
}

func ValidateUpdateFedRoleBindingObject(oldObj, newObj *rbacv1.RoleBinding) error {
	if err := ValidateUpdateK8sObject(oldObj, newObj, new(rbac.RoleBinding), new(rbac.RoleBinding), func(newObj, oldObj interface{}) field.ErrorList {
		rb := newObj.(*rbac.RoleBinding)
		oldRb := oldObj.(*rbac.RoleBinding)
		allErrs := validation.ValidateRoleBinding(rb)
		if oldRb.RoleRef != rb.RoleRef {
			allErrs = append(allErrs, field.Invalid(field.NewPath("roleRef"), rb.RoleRef, "cannot change roleRef"))
		}
		return allErrs
	}); err != nil {
		return errors.Wrap(err, "ValidateUpdateRoleBindingObject")
	}
	return nil
}

func (obj *SFedRoleBinding) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.FedRoleBindingUpdateInput) (*api.FedRoleBindingUpdateInput, error) {
	bInput, err := obj.SFedNamespaceResource.ValidateUpdateData(ctx, userCred, query, &input.FedNamespaceResourceUpdateInput)
	if err != nil {
		return nil, err
	}
	input.FedNamespaceResourceUpdateInput = *bInput
	objMeta := obj.GetK8sObjectMeta()
	oldObj := obj.Spec.ToRoleBinding(objMeta)
	newObj := input.ToRoleBinding(objMeta)
	if err := ValidateUpdateRoleBindingObject(oldObj, newObj); err != nil {
		return nil, err
	}
	return input, nil
}
