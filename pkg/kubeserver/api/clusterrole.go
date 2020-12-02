package api

import (
	rbac "k8s.io/api/rbac/v1"
)

type ClusterRoleCreateInput struct {
	ClusterResourceCreateInput
	Rules []rbac.PolicyRule `json:"rules"`
}

func (input ClusterRoleCreateInput) ToClusterRole() *rbac.ClusterRole {
	objMeta := input.ToObjectMeta()
	return &rbac.ClusterRole{
		ObjectMeta: objMeta,
		Rules:      input.Rules,
	}
}

type ClusterRoleUpdateInput struct {
	ClusterResourceUpdateInput
	Rules []rbac.PolicyRule `json:"rules"`
}
