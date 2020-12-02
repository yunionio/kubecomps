package api

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/jsonutils"
)

type FederatedNamespaceSpec struct {
	Template NamespaceTemplate `json:"template"`
}

func (spec *FederatedNamespaceSpec) String() string {
	return jsonutils.Marshal(spec).String()
}

func (spec *FederatedNamespaceSpec) IsZero() bool {
	if spec == nil {
		return true
	}
	return false
}

func (spec *FederatedNamespaceSpec) ToNamespace(objMeta metav1.ObjectMeta) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: objMeta,
		Spec:       spec.Template.Spec,
	}
}

type NamespaceTemplate struct {
	Spec corev1.NamespaceSpec `json:"spec"`
}

type FederatedNamespaceCreateInput struct {
	FederatedResourceCreateInput
	Spec FederatedNamespaceSpec `json:"spec"`
}

func (input FederatedNamespaceCreateInput) ToNamespace() *corev1.Namespace {
	objMeta := input.ToObjectMeta()
	return input.Spec.ToNamespace(objMeta)
}

type FederatedNamespaceAttachClusterInput struct {
	FederatedResourceJointClusterInput
}

type FederatedNamespaceDetachClusterInput struct {
	FederatedResourceJointClusterInput
}

type FederatedNamespaceDetails struct {
	FederatedResourceDetails
}

type FederatedNamespaceClusterDetails struct {
	FedJointClusterResourceDetails
}

type FederatedNamespaceClusterListInput struct {
	FedJointClusterListInput
}

type FedNamespaceUpdateInput struct {
	FedResourceUpdateInput
	Spec FederatedNamespaceSpec `json:"spec"`
}

func (input FedNamespaceUpdateInput) ToNamespace(objMeta metav1.ObjectMeta) *corev1.Namespace {
	return input.Spec.ToNamespace(objMeta)
}
