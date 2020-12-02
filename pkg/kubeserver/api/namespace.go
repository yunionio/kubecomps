package api

import (
	"k8s.io/api/core/v1"
)

// Namespace is a presentation layer view of Kubernetes namespaces. This means it is namespace plus
// additional augmented data we can get from other sources.
type Namespace struct {
	ObjectMeta
	TypeMeta

	Phase v1.NamespacePhase `json:"status"`
}

// NamespaceDetail is a presentation layer view of Kubernetes Namespace resource. This means it is Namespace plus
// additional augmented data we can get from other sources.
type NamespaceDetail struct {
	Namespace

	// Events is list of events associated to the namespace.
	Events []*Event `json:"events"`

	// ResourceQuotaList is list of resource quotas associated to the namespace
	ResourceQuotas []*ResourceQuotaDetail `json:"resourceQuotas"`

	// ResourceLimits is list of limit ranges associated to the namespace
	ResourceLimits []*LimitRange `json:"limitRanges"`
}

type NamespaceSpec struct {
	v1.NamespaceSpec
}

type NamespaceCreateInput struct {
	K8sClusterResourceCreateInput
}

type NamespaceCreateInputV2 struct {
	ClusterResourceCreateInput
	Spec *NamespaceSpec `json:"spec"`
}

func (input NamespaceCreateInputV2) ToNamespace() *v1.Namespace {
	return &v1.Namespace{
		ObjectMeta: input.ToObjectMeta(),
		Spec:       input.Spec.NamespaceSpec,
	}
}

type NamespaceListInput struct {
	ClusterResourceListInput
}

type NamespaceDetailV2 struct {
	ClusterResourceDetail
}
