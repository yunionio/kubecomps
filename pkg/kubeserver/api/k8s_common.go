package api

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/onecloud/pkg/apis"
)

type K8sClusterResourceGetInput struct {
	// required: true
	Cluster string `json:"cluster"`
}

type K8sNamespaceResourceGetInput struct {
	K8sClusterResourceGetInput
	// required: true
	Namespace string `json:"namespace"`
}

type K8sClusterResourceCreateInput struct {
	// required: true
	Cluster string `json:"cluster"`
	// required: true
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

func (input K8sClusterResourceCreateInput) ToObjectMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:        input.Name,
		Labels:      input.Labels,
		Annotations: input.Annotations,
	}
}

type K8sNamespaceResourceCreateInput struct {
	K8sClusterResourceCreateInput
	// required: true
	Namespace string `json:"namespace"`
}

func (input K8sNamespaceResourceCreateInput) ToObjectMeta() metav1.ObjectMeta {
	objMeta := input.K8sClusterResourceCreateInput.ToObjectMeta()
	objMeta.Namespace = input.Namespace
	return objMeta
}

type K8sResourceCreateInput struct {
	apis.StatusDomainLevelResourceCreateInput
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

func (input K8sResourceCreateInput) ToObjectMeta() metav1.ObjectMeta {
	om := metav1.ObjectMeta{
		Name:        input.Name,
		Labels:      input.Labels,
		Annotations: input.Annotations,
	}
	if om.Labels == nil {
		om.Labels = map[string]string{}
	}
	if om.Annotations == nil {
		om.Annotations = map[string]string{}
	}
	return om
}

type K8sResourceUpdateInput struct {
	apis.StatusDomainLevelResourceBaseUpdateInput
}
