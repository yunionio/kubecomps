package api

import (
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/jsonutils"
)

type FederatedRoleBindingCreateInput struct {
	FederatedNamespaceResourceCreateInput
	Spec *FederatedRoleBindingSpec `json:"spec"`
}

type FederatedRoleBindingSpec struct {
	Template RoleBindingTemplate `json:"template"`
}

func (spec *FederatedRoleBindingSpec) String() string {
	return jsonutils.Marshal(spec).String()
}

func (spec *FederatedRoleBindingSpec) IsZero() bool {
	if spec == nil {
		return true
	}
	return false
}

func (spec *FederatedRoleBindingSpec) ToRoleBinding(objMeta metav1.ObjectMeta) *rbac.RoleBinding {
	return &rbac.RoleBinding{
		ObjectMeta: objMeta,
		Subjects:   spec.Template.Subjects,
		RoleRef:    spec.Template.RoleRef,
	}
}

type RoleBindingTemplate struct {
	Subjects []rbac.Subject `json:"subjects,omitempty"`
	RoleRef  rbac.RoleRef   `json:"roleRef"`
}

func (input FederatedRoleBindingCreateInput) ToRoleBinding(namespace string) *rbac.RoleBinding {
	return input.Spec.ToRoleBinding(input.ToObjectMeta(namespace))
}

type FedRoleBindingUpdateInput struct {
	FedNamespaceResourceUpdateInput
	Spec *FederatedRoleBindingSpec `json:"spec"`
}

func (input FedRoleBindingUpdateInput) ToRoleBinding(objMeta metav1.ObjectMeta) *rbac.RoleBinding {
	return input.Spec.ToRoleBinding(objMeta)
}
