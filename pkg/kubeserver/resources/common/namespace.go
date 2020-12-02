package common

import (
	api "k8s.io/api/core/v1"
)

type NamespaceQuery struct {
	namespaces []string
}

// NewSameNamespaceQuery creates new namespace query that queries single namespace.
func NewSameNamespaceQuery(namespace string) *NamespaceQuery {
	return &NamespaceQuery{[]string{namespace}}
}

func NewNamespaceQuery(namespaces ...string) *NamespaceQuery {
	return &NamespaceQuery{namespaces}
}

func (n *NamespaceQuery) Matches(namespace string) bool {
	if len(n.namespaces) == 0 {
		return true
	}

	for _, ns := range n.namespaces {
		if namespace == ns {
			return true
		}
	}
	return false
}

func (n *NamespaceQuery) ToRequestParam() string {
	if len(n.namespaces) == 1 {
		return n.namespaces[0]
	}
	return api.NamespaceAll
}
