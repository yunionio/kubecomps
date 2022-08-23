package api

import (
	extensions "k8s.io/api/extensions/v1beta1"
	"k8s.io/kubernetes/pkg/apis/networking"
)

type Ingress struct {
	ObjectMeta
	TypeMeta

	// External endpoints of this ingress.
	Endpoints []Endpoint `json:"endpoints"`
}

// IngressDetail API resource provides mechanisms to inject containers with configuration data while keeping
// containers agnostic of Kubernetes
type IngressDetail struct {
	Ingress

	// TODO: replace this with UI specific fields.
	// Spec is the desired state of the Ingress.
	Spec extensions.IngressSpec `json:"spec"`

	// Status is the current state of the Ingress.
	Status extensions.IngressStatus `json:"status"`
}

type IngressDetailV2 struct {
	NamespaceResourceDetail
	// External endpoints of this ingress.
	Endpoints []Endpoint `json:"endpoints,allowempty"`

	// TODO: replace this with UI specific fields.
	// Spec is the desired state of the Ingress.
	Spec extensions.IngressSpec `json:"spec"`

	// Status is the current state of the Ingress.
	Status extensions.IngressStatus `json:"status"`
}

type IngressDetailNew struct {
	NamespaceResourceDetail
	// External endpoints of this ingress.
	Endpoints []Endpoint `json:"endpoints,allowempty"`

	// TODO: replace this with UI specific fields.
	// Spec is the desired state of the Ingress.
	Spec networking.IngressSpec `json:"spec"`

	// Status is the current state of the Ingress.
	Status networking.IngressStatus `json:"status"`
}

type IngressCreateInput struct {
	K8sNamespaceResourceCreateInput
	extensions.IngressSpec
}

type IngressCreateInputV2 struct {
	NamespaceResourceCreateInput
	extensions.IngressSpec
}

type IngressCreateInputNew struct {
	NamespaceResourceCreateInput
	networking.IngressSpec
}
