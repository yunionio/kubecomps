package api

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	ClusterResourceStatusCreating   = "creating"
	ClusterResourceStatusCreateFail = "create_fail"
	ClusterResourceStatusUpdating   = "updating"
	ClusterResourceStatusUpdateFail = "update_fail"
	ClusterResourceStatusDeleting   = "deleting"
	ClusterResourceStatusDeleteFail = "delete_fail"
	ClusterResourceStatusSyncing    = "syncing"
	ClusterResourceStatusSyncFail   = "sync_fail"
	ClusterResourceStatusActive     = "active"
)

type ClusterResourceCreateInput struct {
	K8sResourceCreateInput
	// 集群Id
	ClusterId string `json:"cluster_id"`

	// swagger:ignore
	// Deprecated
	Cluster string `json:"cluster" yunion-deprecated-by:"cluster_id"`
}

type ClusterResourceListInput struct {
	apis.StatusDomainLevelResourceListInput
	System *bool `json:"system"`

	// Deprecated
	Cluster string `json:"cluster" yunion-deprecated-by:"cluster_id"`
	// 集群名称
	ClusterId string `json:"cluster_id"`
	// Deprecated
	Account string `json:"account" yunion-deprecated-by:"cluster_id"`
	// Deprecated
	AccountId string `json:"account_id" yunion-deprecated-by:"cluster_id"`
}

type ClusterResourceUpdateInput struct {
	apis.StatusDomainLevelResourceBaseUpdateInput

	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type ClusterK8SResourceMetaDetail struct {
	metav1.TypeMeta
	// An opaque value that represents the internal version of this object that can
	// be used by clients to determine when objects have changed. May be used for optimistic
	// concurrency, change detection, and the watch operation on a resource or set of resources.
	// Clients must treat these values as opaque and passed unmodified back to the server.
	// They may only be valid for a particular resource or set of resources.
	//
	// Populated by the system.
	// Read-only.
	// Value must be treated as opaque by clients and .
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency
	// +optional
	ResourceVersion string `json:"resourceVersion,omitempty"`

	// A sequence number representing a specific generation of the desired state.
	// Populated by the system. Read-only.
	// +optional
	Generation int64 `json:"generation,omitempty"`

	// CreationTimestamp is a timestamp representing the server time when this object was
	// created. It is not guaranteed to be set in happens-before order across separate operations.
	// Clients may not set this value. It is represented in RFC3339 form and is in UTC.
	//
	// Populated by the system.
	// Read-only.
	// Null for lists.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	CreationTimestamp time.Time `json:"creationTimestamp"`

	// DeletionTimestamp is RFC 3339 date and time at which this resource will be deleted. This
	// field is set by the server when a graceful deletion is requested by the user, and is not
	// directly settable by a client. The resource is expected to be deleted (no longer visible
	// from resource lists, and not reachable by name) after the time in this field, once the
	// finalizers list is empty. As long as the finalizers list contains items, deletion is blocked.
	// Once the deletionTimestamp is set, this value may not be unset or be set further into the
	// future, although it may be shortened or the resource may be deleted prior to this time.
	// For example, a user may request that a pod is deleted in 30 seconds. The Kubelet will react
	// by sending a graceful termination signal to the containers in the pod. After that 30 seconds,
	// the Kubelet will send a hard termination signal (SIGKILL) to the container and after cleanup,
	// remove the pod from the API. In the presence of network partitions, this object may still
	// exist after this timestamp, until an administrator or automated process can determine the
	// resource is fully terminated.
	// If not set, graceful deletion of the object has not been requested.
	//
	// Populated by the system when a graceful deletion is requested.
	// Read-only.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	DeletionTimestamp *time.Time `json:"deletionTimestamp,omitempty"`

	// Number of seconds allowed for this object to gracefully terminate before
	// it will be removed from the system. Only set when deletionTimestamp is also set.
	// May only be shortened.
	// Read-only.
	// +optional
	DeletionGracePeriodSeconds *int64 `json:"deletionGracePeriodSeconds,omitempty"`

	// Map of string keys and values that can be used to organize and categorize
	// (scope and select) objects. May match selectors of replication controllers
	// and services.
	// More info: http://kubernetes.io/docs/user-guide/labels
	// +optional
	Labels map[string]string `json:"labels,omitempty" protobuf:"bytes,11,rep,name=labels"`

	// Annotations is an unstructured key value map stored with a resource that may be
	// set by external tools to store and retrieve arbitrary metadata. They are not
	// queryable and should be preserved when modifying objects.
	// More info: http://kubernetes.io/docs/user-guide/annotations
	// +optional
	Annotations map[string]string `json:"annotations,omitempty" protobuf:"bytes,12,rep,name=annotations"`

	// List of objects depended by this object. If ALL objects in the list have
	// been deleted, this object will be garbage collected. If this object is managed by a controller,
	// then an entry in this list will point to this controller, with the controller field set to true.
	// There cannot be more than one managing controller.
	// +optional
	// +patchMergeKey=uid
	// +patchStrategy=merge
	OwnerReferences []metav1.OwnerReference `json:"ownerReferences,omitempty" patchStrategy:"merge" patchMergeKey:"uid" protobuf:"bytes,13,rep,name=ownerReferences"`

	// Must be empty before the object is deleted from the registry. Each entry
	// is an identifier for the responsible component that will remove the entry
	// from the list. If the deletionTimestamp of the object is non-nil, entries
	// in this list can only be removed.
	// +optional
	// +patchStrategy=merge
	Finalizers []string `json:"finalizers,omitempty" patchStrategy:"merge" protobuf:"bytes,14,rep,name=finalizers"`
}

type ClusterResourceDetail struct {
	apis.StatusDomainLevelResourceDetails
	*ClusterK8SResourceMetaDetail

	ClusterId string `json:"cluster_id"`
	// clusterID is DEPRECATED, use cluster_id
	ClusterID string `json:"clusterID"`
	Cluster   string `json:"cluster"`

	Distribution string `json:"distribution"`
}
