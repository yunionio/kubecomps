package api

// ReplicaSet is a presentation layer view of Kubernetes Replica Set resource. This means
// it is Replica Set plus additional augmented data we can get from other sources
// (like services that target the same pods).
type ReplicaSet struct {
	ObjectMeta
	TypeMeta

	// Aggregate information about pods belonging to this Replica Set.
	Pods PodInfo `json:"pods"`

	// Container images of the Replica Set.
	ContainerImages []ContainerImage `json:"containerImages"`

	// Init Container images of the Replica Set.
	InitContainerImages []ContainerImage `json:"initContainerImages"`
}

type ReplicaSetDetail struct {
	NamespaceResourceDetail

	// Aggregate information about pods belonging to this Replica Set.
	Pods PodInfo `json:"pods"`

	// Container images of the Replica Set.
	ContainerImages []ContainerImage `json:"containerImages"`

	// Init Container images of the Replica Set.
	InitContainerImages []ContainerImage `json:"initContainerImages"`
}
