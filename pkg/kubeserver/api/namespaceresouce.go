package api

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/pkg/errors"
)

type NamespaceResourceCreateInput struct {
	ClusterResourceCreateInput

	// required: true
	// 命名空间
	NamespaceId string `json:"namespace_id"`
	// Namespace should set by backend
	// swagger:ignore
	Namespace string `json:"namespace" yunion-deprecated-by:"namespace_id"`
}

type INamespaceGetter interface {
	GetNamespaceName() (string, error)
}

func (input NamespaceResourceCreateInput) ToObjectMeta(getter INamespaceGetter) (metav1.ObjectMeta, error) {
	objMeta := input.ClusterResourceCreateInput.ToObjectMeta()
	nsName, err := getter.GetNamespaceName()
	if err != nil {
		return metav1.ObjectMeta{}, errors.Wrap(err, "get namespace name")
	}
	objMeta.Namespace = nsName
	return objMeta, nil
}

type NamespaceResourceListInput struct {
	ClusterResourceListInput
	// 命名空间
	Namespace string `json:"namespace"`
}

type NamespaceResourceDetail struct {
	ClusterResourceDetail

	NamespaceId string `json:"namespace_id"`
	Namespace   string `json:"namespace"`

	NamespaceLabels map[string]string `json:"namespace_labels"`
}

type NamespaceResourceUpdateInput struct {
	ClusterResourceUpdateInput
}
