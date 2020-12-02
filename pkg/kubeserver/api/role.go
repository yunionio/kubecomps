package api

import (
	rbac "k8s.io/api/rbac/v1"
)

type RoleCreateInput struct {
	NamespaceResourceCreateInput
	Rules []rbac.PolicyRule `json:"rules"`
}

type nsGetter struct {
	namespace string
}

func newNamespaceGetter(ns string) INamespaceGetter {
	return nsGetter{namespace: ns}
}

func (g nsGetter) GetNamespaceName() (string, error) {
	return g.namespace, nil
}

func (input RoleCreateInput) ToRole(namespaceName string) (*rbac.Role, error) {
	objMeta, err := input.NamespaceResourceCreateInput.ToObjectMeta(newNamespaceGetter(namespaceName))
	if err != nil {
		return nil, err
	}
	return &rbac.Role{
		ObjectMeta: objMeta,
		Rules:      input.Rules,
	}, nil
}

type RoleUpdateInput struct {
	NamespaceResourceUpdateInput
	Rules []rbac.PolicyRule `json:"rules"`
}
