package api

import (
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/jsonutils"
)

type FederatedRoleSpec struct {
	Template RoleTemplate `json:"template"`
}

type RoleTemplate struct {
	Rules []rbac.PolicyRule `json:"rules"`
}

func (spec *FederatedRoleSpec) String() string {
	return jsonutils.Marshal(spec).String()
}

func (spec *FederatedRoleSpec) IsZero() bool {
	if spec == nil {
		return true
	}
	return false
}

func (spec *FederatedRoleSpec) ToRole(objMeta metav1.ObjectMeta) *rbac.Role {
	return &rbac.Role{
		ObjectMeta: objMeta,
		Rules:      spec.Template.Rules,
	}
}

type FedRoleCreateInput struct {
	FederatedNamespaceResourceCreateInput
	Spec *FederatedRoleSpec `json:"spec"`
}

func (input FedRoleCreateInput) ToRole(namespace string) *rbac.Role {
	return input.Spec.ToRole(input.ToObjectMeta(namespace))
}

type FedRoleUpdateInput struct {
	FedNamespaceResourceUpdateInput
	Spec *FederatedRoleSpec `json:"spec"`
}

func (input FedRoleUpdateInput) ToRole(objMeta metav1.ObjectMeta) *rbac.Role {
	return input.Spec.ToRole(objMeta)
}
