package api

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type KindName = string

const (
	KindNameConfigMap               KindName = "ConfigMap"
	KindNameDaemonSet               KindName = "DaemonSet"
	KindNameDeployment              KindName = "Deployment"
	KindNameEvent                   KindName = "Event"
	KindNameHorizontalPodAutoscaler KindName = "HorizontalPodAutoscaler"
	KindNameIngress                 KindName = "Ingress"
	KindNameJob                     KindName = "Job"
	KindNameCronJob                 KindName = "CronJob"
	KindNameNamespace               KindName = "Namespace"
	KindNameNode                    KindName = "Node"
	KindNamePersistentVolumeClaim   KindName = "PersistentVolumeClaim"
	KindNamePersistentVolume        KindName = "PersistentVolume"
	KindNamePod                     KindName = "Pod"
	KindNameReplicaSet              KindName = "ReplicaSet"
	KindNameSecret                  KindName = "Secret"
	KindNameService                 KindName = "Service"
	KindNameStatefulSet             KindName = "StatefulSet"
	KindNameEndpoint                KindName = "Endpoints"
	KindNameStorageClass            KindName = "StorageClass"
	KindNameRole                    KindName = "Role"
	KindNameRoleBinding             KindName = "RoleBinding"
	KindNameClusterRole             KindName = "ClusterRole"
	KindNameClusterRoleBinding      KindName = "ClusterRoleBinding"
	KindNameServiceAccount          KindName = "ServiceAccount"
	KindNameLimitRange              KindName = "LimitRange"
	KindNameResourceQuota           KindName = "ResourceQuota"

	// onecloud service operator native kind
	KindNameVirtualMachine          KindName = "VirtualMachine"
	KindNameAnsiblePlaybook         KindName = "AnsiblePlaybook"
	KindNameAnsiblePlaybookTemplate KindName = "AnsiblePlaybookTemplate"
)

const (
	// kubernetes native resource
	ResourceNameConfigMap               string = "configmaps"
	ResourceNameDaemonSet               string = "daemonsets"
	ResourceNameDeployment              string = "deployments"
	ResourceNameEvent                   string = "events"
	ResourceNameHorizontalPodAutoscaler string = "horizontalpodautoscalers"
	ResourceNameIngress                 string = "ingresses"
	ResourceNameJob                     string = "jobs"
	ResourceNameCronJob                 string = "cronjobs"
	ResourceNameNamespace               string = "namespaces"
	ResourceNameNode                    string = "nodes"
	ResourceNamePersistentVolumeClaim   string = "persistentvolumeclaims"
	ResourceNamePersistentVolume        string = "persistentvolumes"
	ResourceNamePod                     string = "pods"
	ResourceNameReplicaSet              string = "replicasets"
	ResourceNameSecret                  string = "secrets"
	ResourceNameService                 string = "services"
	ResourceNameStatefulSet             string = "statefulsets"
	ResourceNameEndpoint                string = "endpoints"
	ResourceNameStorageClass            string = "storageclasses"
	ResourceNameRole                    string = "roles"
	ResourceNameRoleBinding             string = "rolebindings"
	ResourceNameClusterRole             string = "clusterroles"
	ResourceNameClusterRoleBinding      string = "clusterrolebindings"
	ResourceNameServiceAccount          string = "serviceaccounts"
	ResourceNameLimitRange              string = "limitranges"
	ResourceNameResourceQuota           string = "resourcequotas"

	// onecloud service operator resource
	ResourceNameVirtualMachine          string = "virtualmachines"
	ResourceNameAnsiblePlaybook         string = "ansibleplaybooks"
	ResourceNameAnsiblePlaybookTemplate string = "ansibleplaybooktemplates"
)

// ObjectMeta is metadata about an instance of a resource.
type ObjectMeta struct {
	// kubernetes object meta
	metav1.ObjectMeta
	// onecloud cluster meta info
	*ClusterMeta
}

type TypeMeta struct {
	metav1.TypeMeta
}

type ObjectTypeMeta struct {
	ObjectMeta
	TypeMeta
}

func (m *ObjectTypeMeta) SetObjectMeta(meta ObjectMeta) *ObjectTypeMeta {
	m.ObjectMeta = meta
	return m
}

func (m *ObjectTypeMeta) SetTypeMeta(meta TypeMeta) *ObjectTypeMeta {
	m.TypeMeta = meta
	return m
}

type ClusterMeta struct {
	// Onecloud cluster data
	Cluster   string `json:"cluster"`
	ClusterId string `json:"clusterID"`
	// Deprecated
	TenantId  string `json:"tenant_id"`
	ProjectId string `json:"project_id"`
}

func (m ObjectMeta) GetName() string {
	return m.Name
}

func NewClusterMeta(cluster ICluster) *ClusterMeta {
	return &ClusterMeta{
		Cluster:   cluster.GetName(),
		ClusterId: cluster.GetId(),
	}
}

type ICluster interface {
	GetId() string
	GetName() string
}

// NewObjectMeta returns internal endpoint name for the given service properties, e.g.,
// NewObjectMeta creates a new instance of ObjectMeta struct based on K8s object meta.
func NewObjectMeta(k8SObjectMeta metav1.ObjectMeta, cluster ICluster) ObjectMeta {
	return ObjectMeta{
		ObjectMeta:  k8SObjectMeta,
		ClusterMeta: NewClusterMeta(cluster),
	}
}

func NewTypeMeta(typeMeta metav1.TypeMeta) TypeMeta {
	return TypeMeta{typeMeta}
}

type K8SBaseResource struct {
	ObjectMeta `json:"objectMeta"`
	TypeMeta   `json:"typeMeta"`
}

// Event is a single event representation.
type Event struct {
	ObjectMeta
	TypeMeta

	// A human-readable description of the status of related object.
	Message string `json:"message"`

	// Component from which the event is generated.
	// Deprecated
	SourceComponent string `json:"sourceComponent"`

	// Host name on which the event is generated.
	// Deprecated
	SourceHost string `json:"sourceHost"`

	// Reference to a piece of an object, which triggered an event. For example
	// "spec.containers{name}" refers to container within pod with given name, if no container
	// name is specified, for example "spec.containers[2]", then it refers to container with
	// index 2 in this pod.
	// Deprecated
	SubObject string `json:"object"`

	// The number of times this event has occurred.
	Count int32 `json:"count"`

	// The time at which the event was first recorded.
	FirstSeen metav1.Time `json:"firstSeen"`

	// The time at which the most recent occurrence of this event was recorded.
	LastSeen metav1.Time `json:"lastSeen"`

	// Short, machine understandable string that gives the reason
	// for this event being generated.
	Reason string `json:"reason"`

	// Event type (at the moment only normal and warning are supported).
	Type string `json:"type"`

	// The object that this event is about.
	InvolvedObject v1.ObjectReference `json:"involvedObject"`

	// The component reporting this event. Should be a short machine understandable string.
	Source v1.EventSource `json:"source,omitempty"`

	// Data about the Event series this event represents or nil if it's a singleton Event.
	// +optional
	Series *v1.EventSeries `json:"series,omitempty"`

	// What action was taken/failed regarding to the Regarding object.
	// +optional
	Action string `json:"action,omitempty"`

	// Optional secondary object for more complex actions.
	// +optional
	Related *v1.ObjectReference `json:"related,omitempty"`

	// Name of the controller that emitted this Event, e.g. `kubernetes.io/kubelet`.
	// +optional
	ReportingController string `json:"reportingComponent"`

	// ID of the controller instance, e.g. `kubelet-xyzf`.
	// +optional
	ReportingInstance string `json:"reportingInstance"`
}

type ContainerUpdateInput struct {
	// required: true
	Name  string `json:"name"`
	Image string `json:"image,omitempty"`
}

// TODO: K8SNamespaceResourceUpdateInput shouldn't contains in body, fix them in url path
type K8SNamespaceResourceUpdateInput struct {
	K8sClusterResourceCreateInput
	// required: true
	Namespace string `json:"namespace"`
}

type PodTemplateUpdateInput struct {
	InitContainers []ContainerUpdateInput `json:"initContainers,omitempty"`
	Containers     []ContainerUpdateInput `json:"containers,omitempty"`
	RestartPolicy  v1.RestartPolicy       `json:"restartPolicy,omitempty"`
	DNSPolicy      v1.DNSPolicy           `json:"dnsPolicy,omitempty"`
}

type ContainerImage struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}

type ListInputOwner struct {
	OwnerKind string `json:"owner_kind"`
	OwnerName string `json:"owner_name"`
}

func (input ListInputOwner) ShouldDo() bool {
	return input.OwnerKind != "" && input.OwnerName != ""
}

type EventListInput struct {
	ListInputK8SNamespaceBase
	ListInputOwner
}
