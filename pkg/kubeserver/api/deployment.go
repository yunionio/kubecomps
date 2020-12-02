package api

import (
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	DeploymentStatusNewReplicaUpdating    = "NewReplicaUpdating"
	DeploymentStatusOldReplicaTerminating = "OldReplicaTerminating"
	DeploymentStatusAvailableWaiting      = "AvailableWaiting"
	DeploymentStatusRunning               = "Running"
	DeploymentStatusObservedWaiting       = "ObservedWaiting"
)

type DeploymentUpdateInput struct {
	K8SNamespaceResourceUpdateInput
	Replicas *int32 `json:"replicas"`
	PodTemplateUpdateInput
}

// Deployment is a presentation layer view of kubernetes Deployment resource. This means
// it is Deployment plus additional augmented data we can get from other sources
// (like services that target the same pods)
type Deployment struct {
	ObjectMeta
	TypeMeta

	// Aggregate information about pods belonging to this deployment
	Pods PodInfo `json:"podsInfo"`

	Replicas *int32 `json:"replicas"`

	// Container images of the Deployment
	ContainerImages []ContainerImage `json:"containerImages"`

	// Init Container images of deployment
	InitContainerImages []ContainerImage `json:"initContainerImages"`

	Selector map[string]string `json:"selector"`

	DeploymentStatus
}

type DeploymentStatus struct {
	Status string `json:"status"`

	// Number of the pod with ready state.
	ReadyReplicas int64 `json:"readyReplicas"`
	// Number of desired pods
	DesiredReplicas int64 `json:"desiredReplicas"`
	// Total number of non-terminated pods targeted by this deployment that have the desired template spec.
	UpdatedReplicas int64 `json:"updatedReplicas"`
	// Total number of available pods (ready for at least minReadySeconds) targeted by this deployment.
	AvailableReplicas int64 `json:"availableReplicas"`
}

type StatusInfo struct {
	// Total number of desired replicas on the deployment
	Replicas int32 `json:"replicas"`

	// Number of non-terminated pods that have the desired template spec
	Updated int32 `json:"updated"`

	// Number of available pods (ready for at least minReadySeconds)
	// targeted by this deployment
	Available int32 `json:"available"`

	// Total number of unavailable pods targeted by this deployment.
	Unavailable int32 `json:"unavailable"`
}

// RollingUpdateStrategy is behavior of a rolling update. See RollingUpdateDeployment K8s object.
type RollingUpdateStrategy struct {
	MaxSurge       *intstr.IntOrString `json:"maxSurge"`
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable"`
}

// DeploymentDetail is a presentation layer view of Kubernetes Deployment resource.
type DeploymentDetail struct {
	Deployment
	// Detailed information about Pods belonging to this Deployment.
	Pods []*Pod `json:"pods"`

	Services []*Service `json:"services"`

	// Status information on the deployment
	StatusInfo `json:"statusInfo"`

	// The deployment strategy to use to replace existing pods with new ones.
	// Valid options: Recreate, RollingUpdate
	Strategy apps.DeploymentStrategyType `json:"strategy"`

	// Min ready seconds
	MinReadySeconds int32 `json:"minReadySeconds"`

	// Rolling update strategy containing maxSurge and maxUnavailable
	RollingUpdateStrategy *RollingUpdateStrategy `json:"rollingUpdateStrategy,omitempty"`

	// RepliaSets containing old replica sets from the deployment
	OldReplicaSets []*ReplicaSet `json:"oldReplicaSets"`

	// New replica set used by this deployment
	NewReplicaSet *ReplicaSet `json:"newReplicaSet"`

	// Optional field that specifies the number of old Replica Sets to retain to allow rollback.
	RevisionHistoryLimit *int32 `json:"revisionHistoryLimit"`

	// List of events related to this Deployment
	Events []*Event `json:"events"`

	// List of Horizontal Pod AutoScalers targeting this Deployment
	//HorizontalPodAutoscalerList hpa.HorizontalPodAutoscalerList `json:"horizontalPodAutoscalerList"`
}

type DeploymentCreateInput struct {
	// K8sNamespaceResourceCreateInput
	NamespaceResourceCreateInput

	apps.DeploymentSpec

	Service *ServiceCreateOption `json:"service"`
}

func AddObjectMetaDefaultLabel(meta *metav1.ObjectMeta) *metav1.ObjectMeta {
	return AddObjectMetaRunLabel(meta)
}

func AddObjectMetaRunLabel(meta *metav1.ObjectMeta) *metav1.ObjectMeta {
	if len(meta.Labels) == 0 {
		meta.Labels["run"] = meta.GetName()
	}
	return meta
}

func GetSelectorByObjectMeta(meta *metav1.ObjectMeta) *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchLabels: meta.GetLabels(),
	}
}

func (input *DeploymentCreateInput) ToDeployment(namespaceName string) (*apps.Deployment, error) {
	objMeta, err := input.NamespaceResourceCreateInput.ToObjectMeta(newNamespaceGetter(namespaceName))
	if err != nil {
		return nil, err
	}
	objMeta = *AddObjectMetaDefaultLabel(&objMeta)
	input.Selector = GetSelectorByObjectMeta(&objMeta)
	input.Template.ObjectMeta = objMeta
	return &apps.Deployment{
		ObjectMeta: objMeta,
		Spec:       input.DeploymentSpec,
	}, nil
}

type DeploymentListInput struct {
	NamespaceResourceListInput
}

type DeploymentDetailV2 struct {
	NamespaceResourceDetail
	// Aggregate information about pods belonging to this deployment
	Pods     PodInfo `json:"podsInfo"`
	Replicas *int32  `json:"replicas"`
	// Container images of the Deployment
	ContainerImages []ContainerImage `json:"containerImages"`
	// Init Container images of deployment
	InitContainerImages []ContainerImage  `json:"initContainerImages"`
	Selector            map[string]string `json:"selector"`
	DeploymentStatus
	// Rolling update strategy containing maxSurge and maxUnavailable
	RollingUpdateStrategy *RollingUpdateStrategy `json:"rollingUpdateStrategy,omitempty"`
	// The deployment strategy to use to replace existing pods with new ones.
	// Valid options: Recreate, RollingUpdate
	Strategy apps.DeploymentStrategyType `json:"strategy"`
	// Min ready seconds
	MinReadySeconds int32 `json:"minReadySeconds"`
	// Optional field that specifies the number of old Replica Sets to retain to allow rollback.
	RevisionHistoryLimit *int32 `json:"revisionHistoryLimit"`
}
