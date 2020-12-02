package api

import (
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/jsonutils"
)

type FederatedClusterRoleSpec struct {
	Template ClusterRoleTemplate `json:"template"`
}

type ClusterRoleTemplate struct {
	Rules []rbac.PolicyRule `json:"rules"`
}

func (spec *FederatedClusterRoleSpec) String() string {
	return jsonutils.Marshal(spec).String()
}

func (spec *FederatedClusterRoleSpec) IsZero() bool {
	if spec == nil {
		return true
	}
	return false
}

func (spec *FederatedClusterRoleSpec) ToClusterRole(objMeta metav1.ObjectMeta) *rbac.ClusterRole {
	return &rbac.ClusterRole{
		ObjectMeta: objMeta,
		Rules:      spec.Template.Rules,
	}
}

type FederatedClusterRoleCreateInput struct {
	FederatedResourceCreateInput
	Spec *FederatedClusterRoleSpec `json:"spec"`
}

func (input FederatedClusterRoleCreateInput) ToClusterRole() *rbac.ClusterRole {
	return input.Spec.ToClusterRole(input.ToObjectMeta())
}

type FederatedClusterRoleClusterListInput struct {
	FedJointClusterListInput
}

type FedClusterRoleUpdateInput struct {
	FedResourceUpdateInput
	Spec *FederatedClusterRoleSpec `json:"spec"`
}

func (input FedClusterRoleUpdateInput) ToClusterRole(objMeta metav1.ObjectMeta) *rbac.ClusterRole {
	return input.Spec.ToClusterRole(objMeta)
}
