package api

import (
	"k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
)

type ClusterRoleDetail struct {
	ClusterResourceDetail
	Rules []rbac.PolicyRule `json:"rules"`

	AggregationRule *rbac.AggregationRule `json:"aggregationRule,omitempty"`
}

type RoleDetail struct {
	NamespaceResourceDetail
	Rules []rbac.PolicyRule `json:"rules"`
}

type RoleBindingDetail struct {
	NamespaceResourceDetail

	Subjects []rbac.Subject `json:"subjects,omitempty"`

	RoleRef rbac.RoleRef `json:"roleRef"`
}

// ClusterRoleBinding references a ClusterRole, but not contain it.  It can reference a ClusterRole in the global namespace,
// and adds who information via Subject.
type ClusterRoleBindingDetail struct {
	ClusterResourceDetail

	// Subjects holds references to the objects the role applies to.
	// +optional
	Subjects []rbac.Subject `json:"subjects,omitempty"`

	// RoleRef can only reference a ClusterRole in the global namespace.
	// If the RoleRef cannot be resolved, the Authorizer must return an error.
	RoleRef rbac.RoleRef `json:"roleRef"`
}

type ServiceAccountDetail struct {
	NamespaceResourceDetail
	// Secrets is the list of secrets allowed to be used by pods running using this ServiceAccount.
	// More info: https://kubernetes.io/docs/concepts/configuration/secret
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge
	Secrets []v1.ObjectReference `json:"secrets,omitempty"`

	// ImagePullSecrets is a list of references to secrets in the same namespace to use for pulling any images
	// in pods that reference this ServiceAccount. ImagePullSecrets are distinct from Secrets because Secrets
	// can be mounted in the pod, but ImagePullSecrets are only accessed by the kubelet.
	// More info: https://kubernetes.io/docs/concepts/containers/images/#specifying-imagepullsecrets-on-a-pod
	// +optional
	ImagePullSecrets []v1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// AutomountServiceAccountToken indicates whether pods running as this service account should have an API token automatically mounted.
	// Can be overridden at the pod level.
	// +optional
	AutomountServiceAccountToken *bool `json:"automountServiceAccountToken,omitempty"`
}
