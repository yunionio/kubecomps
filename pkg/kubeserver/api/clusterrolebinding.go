package api

import (
	rbac "k8s.io/api/rbac/v1"
)

type ClusterRoleBindingCreateInput struct {
	ClusterResourceCreateInput
	// Subjects holds references to the objects the role applies to.
	// +optional
	Subjects Subjects `json:"subjects,omitempty"`

	// RoleRef can only reference a ClusterRole in the global namespace.
	// If the RoleRef cannot be resolved, the Authorizer must return an error.
	RoleRef RoleRef `json:"roleRef"`
}

func (crb ClusterRoleBindingCreateInput) ToClusterRoleBinding() *rbac.ClusterRoleBinding {
	return &rbac.ClusterRoleBinding{
		ObjectMeta: crb.ToObjectMeta(),
		Subjects:   crb.Subjects,
		RoleRef:    rbac.RoleRef(crb.RoleRef),
	}
}

type ClusterRoleBindingUpdateInput struct {
	ClusterResourceUpdateInput
	Subjects Subjects `json:"subjects,omitempty"`
	RoleRef  RoleRef  `json:"roleRef"`
}
