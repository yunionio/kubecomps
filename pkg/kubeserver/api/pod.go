package api

import (
	"k8s.io/api/core/v1"
)

// Pod is a presentation layer view of Pod resource. This means it is Pod plus additional augmented data
// we can get from other sources (like services that target it).
type Pod struct {
	ObjectMeta
	TypeMeta

	// More info on pod status
	PodStatus

	PodIP string `json:"podIP"`
	// Count of containers restarts
	RestartCount int32 `json:"restartCount"`

	// Pod warning events
	Warnings []*Event `json:"warnings"`

	QOSClass       string      `json:"qosClass"`
	Containers     []Container `json:"containers"`
	InitContainers []Container `json:"initContainers"`

	// Container images of the Deployment
	ContainerImages []ContainerImage `json:"containerImages"`
	// Init Container images of deployment
	InitContainerImages []ContainerImage `json:"initContainerImages"`
}

type PodStatus struct {
	PodStatusV2

	PodPhase        v1.PodPhase         `json:"podPhase"`
	ContainerStates []v1.ContainerState `json:"containerStates"`
}

type PodStatusV2 struct {
	// The aggregate readiness state of this pod for accepting traffic.
	Ready string `json:"ready"`
	// The aggregate status of the containers in this pod.
	Status string `json:"status"`
	// The number of times the containers in this pod have been restarted.
	Restarts int64 `json:"restarts"`
	// Name of the Node this pod runs on
	NodeName string `json:"nodeName"`
	// NominatedNodeName is set only when this pod preempts other pods on the node,
	// but it cannot be scheduled right away as preemption victims receive their graceful termination periods.
	// This field does not guarantee that the pod will be scheduled on this node.
	// Scheduler may decide to place the pod elsewhere if other nodes become available sooner.
	// Scheduler may also decide to give the resources on this node to a higher priority pod that is created after preemption.
	// As a result, this field may be different than PodSpec.nodeName when the pod is scheduled.
	NominatedNodeName string `json:"nominatedNodeName"`
	// If specified, all readiness gates will be evaluated for pod readiness.
	// A pod is ready when all its containers are ready AND all conditions specified
	// in the readiness gates have status equal to \"True\"
	// More info: https://git.k8s.io/enhancements/keps/sig-network/0007-pod-ready%2B%2B.md
	ReadinessGates string `json:"readinessGates"`
}

type PodDetail struct {
	Pod
	Conditions             []*Condition             `json:"conditions"`
	Events                 []*Event                 `json:"events"`
	Persistentvolumeclaims []*PersistentVolumeClaim `json:"persistentVolumeClaims"`
	ConfigMaps             []*ConfigMap             `json:"configMaps"`
	Secrets                []*Secret                `json:"secrets"`
}

type PodDetailV2 struct {
	NamespaceResourceDetail

	// More info on pod status
	PodStatus
	PodIP string `json:"podIP"`
	// Count of containers restarts
	RestartCount int32 `json:"restartCount"`
	// Pod warning events
	Warnings       []*Event    `json:"warnings"`
	QOSClass       string      `json:"qosClass"`
	Containers     []Container `json:"containers"`
	InitContainers []Container `json:"initContainers"`
	// Container images of the Deployment
	ContainerImages []ContainerImage `json:"containerImages"`
	// Init Container images of deployment
	InitContainerImages []ContainerImage `json:"initContainerImages"`
	Conditions          []*Condition     `json:"conditions"`
	/* Events                 []*Event                 `json:"events"`
	 * Persistentvolumeclaims []*PersistentVolumeClaim `json:"persistentVolumeClaims"`
	 * ConfigMaps             []*ConfigMap             `json:"configMaps"`
	 * Secrets                []*Secret                `json:"secrets"`
	 */
}

// Container represents a docker/rkt/etc. container that lives in a pod.
type Container struct {
	// Name of the container.
	Name string `json:"name"`

	// Image URI of the container.
	Image string `json:"image"`

	// List of environment variables.
	Env []EnvVar `json:"env"`

	// Commands of the container
	Commands []string `json:"commands"`

	// Command arguments
	Args []string `json:"args"`
}

// EnvVar represents an environment variable of a container.
type EnvVar struct {
	// Name of the variable.
	Name string `json:"name"`

	// Value of the variable. May be empty if value from is defined.
	Value string `json:"value"`

	// Defined for derived variables. If non-null, the value is get from the reference.
	// Note that this is an API struct. This is intentional, as EnvVarSources are plain struct
	// references.
	ValueFrom *v1.EnvVarSource `json:"valueFrom"`
}

// PodInfo represents aggregate information about controller's pods.
type PodInfo struct {
	// Number of pods that are created.
	Current int32 `json:"current"`

	// Number of pods that are desired.
	Desired *int32 `json:"desired,omitempty"`

	// Number of pods that are currently running.
	Running int32 `json:"running"`

	// Number of pods that are currently waiting.
	Pending int32 `json:"pending"`

	// Number of pods that are failed.
	Failed int32 `json:"failed"`

	// Number of pods that are succeeded.
	Succeeded int32 `json:"succeeded"`

	// Unique warning messages related to pods in this resource.
	Warnings []Event `json:"warnings"`
}

func (info PodInfo) GetStatus() string {
	if info.Current == 0 {
		// delete
		return string(v1.PodPending)
	}
	if info.Failed > 0 {
		return string(v1.PodFailed)
	}
	if info.Pending > 0 {
		return string(v1.PodPending)
	}
	if info.Succeeded == *info.Desired {
		return string(v1.PodSucceeded)
	}
	if info.Running == *info.Desired {
		return string(v1.PodRunning)
	}
	return string(v1.PodUnknown)
}

type PodListInput struct {
	NamespaceResourceListInput
	ListInputOwner
}
