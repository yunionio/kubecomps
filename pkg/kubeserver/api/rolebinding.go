package api

import (
	rbac "k8s.io/api/rbac/v1"

	"yunion.io/x/jsonutils"
)

type Subjects []rbac.Subject

func (ss *Subjects) String() string {
	return jsonutils.Marshal(ss).String()
}

func (ss *Subjects) IsZero() bool {
	if ss == nil {
		return true
	}
	return false
}

type RoleRef rbac.RoleRef

func (rf *RoleRef) String() string {
	return jsonutils.Marshal(rf).String()
}

func (rf *RoleRef) IsZero() bool {
	if rf == nil {
		return true
	}
	return false
}

type RoleBindingCreateInput struct {
	NamespaceResourceCreateInput
	// Subjects holds references to the objects the role applies to.
	// +optional
	Subjects Subjects `json:"subjects,omitempty"`
	// RoleRef can reference a Role in the current namespace or a ClusterRole in the global namespace.
	// If the RoleRef cannot be resolved, the Authorizer must return an error.
	RoleRef RoleRef `json:"roleRef"`
}

func (rb RoleBindingCreateInput) ToRoleBinding(namespaceName string) (*rbac.RoleBinding, error) {
	objMeta, err := rb.ToObjectMeta(newNamespaceGetter(namespaceName))
	if err != nil {
		return nil, err
	}
	return &rbac.RoleBinding{
		ObjectMeta: objMeta,
		Subjects:   rb.Subjects,
		RoleRef:    rbac.RoleRef(rb.RoleRef),
	}, nil
}

type RoleBindingUpdateInput struct {
	NamespaceResourceUpdateInput
	Subjects Subjects `json:"subjects,omitempty"`
	RoleRef  RoleRef  `json:"roleRef"`
}
