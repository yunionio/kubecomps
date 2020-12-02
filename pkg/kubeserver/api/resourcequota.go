package api

import (
	"k8s.io/api/core/v1"
)

type ResourceQuota struct {
	ObjectMeta
	TypeMeta

	v1.ResourceQuotaSpec
}

// ResourceQuotaDetail provides the presentation layer view of Kubernetes Resource Quotas resource.
type ResourceQuotaDetail struct {
	ResourceQuota

	// StatusList is a set of (resource name, Used, Hard) tuple.
	StatusList map[v1.ResourceName]ResourceStatus `json:"statuses,omitempty"`
}

type ResourceQuotaDetailV2 struct {
	NamespaceResourceDetail
	v1.ResourceQuotaSpec

	// StatusList is a set of (resource name, Used, Hard) tuple.
	StatusList map[v1.ResourceName]ResourceStatus `json:"statuses,omitempty"`
}

// ResourceStatus provides the status of the resource defined by a resource quota.
type ResourceStatus struct {
	Used string `json:"used,omitempty"`
	Hard string `json:"hard,omitempty"`
}
