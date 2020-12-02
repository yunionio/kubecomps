package api

import (
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/jsonutils"
)

type FederatedClusterRoleBindingCreateInput struct {
	FederatedResourceCreateInput
	Spec *FederatedClusterRoleBindingSpec `json:"spec"`
}

func (input FederatedClusterRoleBindingCreateInput) ToClusterRoleBinding() *rbac.ClusterRoleBinding {
	return input.Spec.ToClusterRoleBinding(input.ToObjectMeta())
}

type FederatedClusterRoleBindingSpec struct {
	Template ClusterRoleBindingTemplate `json:"template"`
}

func (spec *FederatedClusterRoleBindingSpec) String() string {
	return jsonutils.Marshal(spec).String()
}

func (spec *FederatedClusterRoleBindingSpec) IsZero() bool {
	return spec == nil
}

func (spec FederatedClusterRoleBindingSpec) ToClusterRoleBinding(objMeta metav1.ObjectMeta) *rbac.ClusterRoleBinding {
	return &rbac.ClusterRoleBinding{
		ObjectMeta: objMeta,
		RoleRef:    spec.Template.RoleRef,
		Subjects:   spec.Template.Subjects,
	}
}

type ClusterRoleBindingTemplate struct {
	Subjects []rbac.Subject `json:"subjects,omitempty"`
	// RoleRef can only reference a FederatedClusterRole in the global namespace.
	RoleRef rbac.RoleRef `json:"roleRef"`
}

type FedClusterRoleBindingClusterListInput struct {
	FedJointClusterListInput
}

type FedClusterRoleBindingUpdateInput struct {
	FedResourceUpdateInput
	Spec *FederatedClusterRoleBindingSpec `json:"spec"`
}

func (input FedClusterRoleBindingUpdateInput) ToClusterRoleBinding(objMeta metav1.ObjectMeta) *rbac.ClusterRoleBinding {
	return input.Spec.ToClusterRoleBinding(objMeta)
}
