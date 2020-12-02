package api

import (
	apps "k8s.io/api/apps/v1"
)

const (
	StatefulSetStatusObservedWaiting    = "ObservedWaiting"
	StatefulSetStatusPodReadyWaiting    = "PodReadyWaiting"
	StatefulSetStatusNewReplicaUpdating = "NewReplicaUpdating"
	StatefulSetStatusUpdateWaiting      = "UpdateWaiting"
)

// StatefulSet is a presentation layer view of Kubernetes Stateful Set resource. This means it is
// Stateful Set plus additional augmented data we can get from other sources (like services that
// target the same pods).
type StatefulSet struct {
	ObjectMeta
	TypeMeta

	// Aggregate information about pods belonging to this Pet Set.
	Pods PodInfo `json:"podsInfo"`

	Replicas *int32 `json:"replicas"`

	// Container images of the Stateful Set.
	ContainerImages []ContainerImage `json:"containerImages"`

	// Init container images of the Stateful Set.
	InitContainerImages []ContainerImage  `json:"initContainerImages"`
	Selector            map[string]string `json:"selector"`

	StatefulSetStatus
}

type StatefulSetStatus struct {
	Status string `json:"status"`
}

// StatefulSetDetail is a presentation layer view of Kubernetes Stateful Set resource. This means it is Stateful
// Set plus additional augmented data we can get from other sources (like services that target the same pods).
type StatefulSetDetail struct {
	StatefulSet
	PodList  []*Pod     `json:"pods"`
	Events   []*Event   `json:"events"`
	Services []*Service `json:"services"`
}

type StatefulSetDetailV2 struct {
	NamespaceResourceDetail

	// Aggregate information about pods belonging to this Pet Set.
	Pods     PodInfo `json:"podsInfo"`
	Replicas *int32  `json:"replicas"`
	// Container images of the Stateful Set.
	ContainerImages []ContainerImage `json:"containerImages"`
	// Init container images of the Stateful Set.
	InitContainerImages []ContainerImage  `json:"initContainerImages"`
	Selector            map[string]string `json:"selector"`
	StatefulSetStatus
	/*
	 * PodList  []*Pod     `json:"pods"`
	 * Events   []*Event   `json:"events"`
	 * Services []*Service `json:"services"`
	 */
}

type StatefulsetCreateInput struct {
	NamespaceResourceCreateInput
	apps.StatefulSetSpec

	Service *ServiceCreateOption `json:"service"`
}

func (input StatefulsetCreateInput) ToStatefulset(namespaceName string) (*apps.StatefulSet, error) {
	objMeta, err := input.NamespaceResourceCreateInput.ToObjectMeta(newNamespaceGetter(namespaceName))
	if err != nil {
		return nil, err
	}
	objMeta = *AddObjectMetaDefaultLabel(&objMeta)
	input.Template.ObjectMeta = objMeta
	input.Selector = GetSelectorByObjectMeta(&objMeta)
	input.ServiceName = objMeta.GetName()
	return &apps.StatefulSet{
		ObjectMeta: objMeta,
		Spec:       input.StatefulSetSpec,
	}, nil
}

type StatefulsetUpdateInput struct {
	K8SNamespaceResourceUpdateInput
	Replicas *int32 `json:"replicas"`
	PodTemplateUpdateInput
}
