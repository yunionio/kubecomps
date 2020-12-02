package api

import (
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DaemonSetStatusObservedWaiting = "ObservedWaiting"
	DaemonSetStatusPodReadyWaiting = "PodReadyWaiting"
	DaemonSetStatusUpdateWaiting   = "UpdateWaiting"
)

// DaemonSet plus zero or more Kubernetes services that target the Daemon Set.
type DaemonSet struct {
	ObjectMeta
	TypeMeta

	// Aggregate information about pods belonging to this deployment
	PodInfo PodInfo `json:"podsInfo"`

	ContainerImages     []ContainerImage  `json:"containerImages"`
	InitContainerImages []ContainerImage  `json:"initContainerImages"`
	Selector            *v1.LabelSelector `json:"labelSelector"`

	DaemonSetStatus
}

type DaemonSetStatus struct {
	Status string `json:"status"`
}

type DaemonSetDetailV2 struct {
	NamespaceResourceDetail
	// Aggregate information about pods belonging to this deployment
	PodInfo             PodInfo           `json:"podsInfo"`
	ContainerImages     []ContainerImage  `json:"containerImages"`
	InitContainerImages []ContainerImage  `json:"initContainerImages"`
	Selector            *v1.LabelSelector `json:"labelSelector"`
	DaemonSetStatus
}

type DaemonSetDetail struct {
	DaemonSet

	Events []*Event `json:"events"`
}

type DaemonSetCreateInput struct {
	NamespaceResourceCreateInput

	apps.DaemonSetSpec
	Service *ServiceCreateOption `json:"service"`
}

func (input DaemonSetCreateInput) ToDaemonset(namespaceName string) (*apps.DaemonSet, error) {
	objMeta, err := input.NamespaceResourceCreateInput.ToObjectMeta(newNamespaceGetter(namespaceName))
	if err != nil {
		return nil, err
	}
	objMeta = *AddObjectMetaDefaultLabel(&objMeta)
	input.Template.ObjectMeta = objMeta
	input.Selector = GetSelectorByObjectMeta(&objMeta)
	return &apps.DaemonSet{
		ObjectMeta: objMeta,
		Spec:       input.DaemonSetSpec,
	}, nil
}

type DaemonSetUpdateInput struct {
	K8SNamespaceResourceUpdateInput
	PodTemplateUpdateInput
}
