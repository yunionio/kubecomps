package api

import (
	"k8s.io/api/core/v1"
)

const (
	NodeStatusReady    = "Ready"
	NodeStatusNotReady = "NotReady"
)

// NodeAllocatedResources describes node allocated resources.
type NodeAllocatedResources struct {
	// CPURequests is number of allocated milicores.
	CPURequests int64 `json:"cpuRequests"`

	// CPURequestsFraction is a fraction of CPU, that is allocated.
	CPURequestsFraction float64 `json:"cpuRequestsFraction"`

	// CPULimits is defined CPU limit.
	CPULimits int64 `json:"cpuLimits"`

	// CPULimitsFraction is a fraction of defined CPU limit, can be over 100%, i.e.
	// overcommitted.
	CPULimitsFraction float64 `json:"cpuLimitsFraction"`

	// CPUCapacity is specified node CPU capacity in milicores.
	CPUCapacity int64 `json:"cpuCapacity"`

	// MemoryRequests is a fraction of memory, that is allocated.
	MemoryRequests int64 `json:"memoryRequests"`

	// MemoryRequestsFraction is a fraction of memory, that is allocated.
	MemoryRequestsFraction float64 `json:"memoryRequestsFraction"`

	// MemoryLimits is defined memory limit.
	MemoryLimits int64 `json:"memoryLimits"`

	// MemoryLimitsFraction is a fraction of defined memory limit, can be over 100%, i.e.
	// overcommitted.
	MemoryLimitsFraction float64 `json:"memoryLimitsFraction"`

	// MemoryCapacity is specified node memory capacity in bytes.
	MemoryCapacity int64 `json:"memoryCapacity"`

	// AllocatedPods in number of currently allocated pods on the node.
	AllocatedPods int `json:"allocatedPods"`

	// PodCapacity is maximum number of pods, that can be allocated on the node.
	PodCapacity int64 `json:"podCapacity"`

	// PodFraction is a fraction of pods, that can be allocated on given node.
	PodFraction float64 `json:"podFraction"`
}

// Node is a presentation layer view of Kubernetes nodes. This means it is node plus additional
// augmented data we can get from other sources.
type Node struct {
	ObjectMeta
	TypeMeta
	Ready              bool                   `json:"ready"`
	AllocatedResources NodeAllocatedResources `json:"allocatedResources"`
	// Addresses is a list of addresses reachable to the node. Queried from cloud provider, if available.
	Address []v1.NodeAddress `json:"addresses,omitempty"`
	// Set of ids/uuids to uniquely identify the node.
	NodeInfo v1.NodeSystemInfo `json:"nodeInfo"`
	// Taints
	Taints []v1.Taint `json:"taints,omitempty"`
	// Unschedulable controls node schedulability of new pods. By default node is schedulable.
	Unschedulable bool `json:"unschedulable"`
}

// NodeDetail is a presentation layer view of Kubernetes Node resource. This means it is Node plus
// additional augmented data we can get from other sources.
type NodeDetail struct {
	Node

	// NodePhase is the current lifecycle phase of the node.
	Phase v1.NodePhase `json:"status"`

	// PodCIDR represents the pod IP range assigned to the node.
	PodCIDR string `json:"podCIDR"`

	// ID of the node assigned by the cloud provider.
	ProviderID string `json:"providerID"`

	// Conditions is an array of current node conditions.
	Conditions []*Condition `json:"conditions"`

	// Container images of the node.
	ContainerImages []string `json:"containerImages"`

	// PodList contains information about pods belonging to this node.
	PodList []*Pod `json:"pods"`

	// Events is list of events associated to the node.
	EventList []*Event `json:"events"`

	// Metrics collected for this resource
	//Metrics []metricapi.Metric `json:"metrics"`
}

type ListInputNode struct {
	ListInputK8SClusterBase
}

type NodeCreateInput struct {
	ClusterResourceCreateInput
	// TODO: implement create
}

type NodeListInput struct {
	ClusterResourceListInput
}

type NodeDetailV2 struct {
	ClusterResourceDetail
	Ready              bool                   `json:"ready"`
	AllocatedResources NodeAllocatedResources `json:"allocatedResources"`
	// Addresses is a list of addresses reachable to the node. Queried from cloud provider, if available.
	Address []v1.NodeAddress `json:"addresses,omitempty"`
	// Set of ids/uuids to uniquely identify the node.
	NodeInfo v1.NodeSystemInfo `json:"nodeInfo"`
	// Taints
	Taints []v1.Taint `json:"taints,omitempty"`
	// Unschedulable controls node schedulability of new pods. By default node is schedulable.
	Unschedulable bool `json:"unschedulable"`

	// NodeDetail extra fields
	// NodePhase is the current lifecycle phase of the node.
	Phase v1.NodePhase `json:"status"`

	// PodCIDR represents the pod IP range assigned to the node.
	PodCIDR string `json:"podCIDR"`

	// ID of the node assigned by the cloud provider.
	ProviderID string `json:"providerID"`

	// Conditions is an array of current node conditions.
	Conditions []*Condition `json:"conditions"`

	// Container images of the node.
	ContainerImages []string `json:"containerImages"`

	// PodList contains information about pods belonging to this node.
	Pods []*Pod `json:"pods"`

	// Events is list of events associated to the node.
	Events []*Event `json:"events"`

	// Metrics collected for this resource
	//Metrics []metricapi.Metric `json:"metrics"`
}
