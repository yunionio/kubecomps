package apis

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"

	"yunion.io/x/pkg/util/sets"
)

type IListMeta interface {
	GetLimit() int
	GetTotal() int
	GetOffset() int
}

// ListMeta describes list of objects, i.e. holds information about pagination options set for the list.
type ListMeta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func (l ListMeta) GetTotal() int {
	return l.Total
}

func (l ListMeta) GetLimit() int {
	return l.Limit
}

func (l ListMeta) GetOffset() int {
	return l.Offset
}

// List of all resource kinds supported by the UI.
const (
	ResourceKindConfigMap               = "configmap"
	ResourceKindDaemonSet               = "daemonset"
	ResourceKindDeployment              = "deployment"
	ResourceKindEvent                   = "event"
	ResourceKindHorizontalPodAutoscaler = "horizontalpodautoscaler"
	ResourceKindIngress                 = "ingress"
	ResourceKindJob                     = "job"
	ResourceKindCronJob                 = "cronjob"
	ResourceKindLimitRange              = "limitrange"
	ResourceKindNamespace               = "namespace"
	ResourceKindNode                    = "node"
	ResourceKindPersistentVolumeClaim   = "persistentvolumeclaim"
	ResourceKindPersistentVolume        = "persistentvolume"
	ResourceKindPod                     = "pod"
	ResourceKindReplicaSet              = "replicaset"
	ResourceKindReplicationController   = "replicationcontroller"
	ResourceKindResourceQuota           = "resourcequota"
	ResourceKindSecret                  = "secret"
	ResourceKindService                 = "service"
	ResourceKindStatefulSet             = "statefulset"
	ResourceKindStorageClass            = "storageclass"
	ResourceKindRbacRole                = "role"
	ResourceKindRbacClusterRole         = "clusterrole"
	ResourceKindRbacRoleBinding         = "rolebinding"
	ResourceKindRbacClusterRoleBinding  = "clusterrolebinding"
	ResourceKindEndpoint                = "endpoint"
)

// IsSelectorMatching returns true when an object with the given selector targets the same
// Resources (or subset) that the tested object with the given selector.
func IsSelectorMatching(labelSelector map[string]string, testedObjectLabels map[string]string) bool {
	// If service has no selectors, then assume it targets different resource.
	if len(labelSelector) == 0 {
		return false
	}
	for label, value := range labelSelector {
		if rsValue, ok := testedObjectLabels[label]; !ok || rsValue != value {
			return false
		}
	}
	return true
}

// IsLabelSelectorMatching returns true when a resource with the given selector targets the same
// Resources(or subset) that a tested object selector with the given selector.
func IsLabelSelectorMatching(selector map[string]string, labelSelector *v1.LabelSelector) bool {
	// If the resource has no selectors, then assume it targets different Pods.
	if len(selector) == 0 {
		return false
	}
	for label, value := range selector {
		if rsValue, ok := labelSelector.MatchLabels[label]; !ok || rsValue != value {
			return false
		}
	}
	return true
}

// ListEverything is a list options used to list all resources without any filtering.
var ListEverything = metaV1.ListOptions{
	LabelSelector: labels.Everything().String(),
	FieldSelector: fields.Everything().String(),
}

// ClientType represents type of client that is used to perform generic operations on resources.
// Different resources belong to different client, i.e. Deployments belongs to extension client
// and StatefulSets to apps client.
type ClientType string

// List of client types supported by the UI.
const (
	ClientTypeDefault           = "restclient"
	ClientTypeExtensionClient   = "extensionclient"
	ClientTypeAppsClient        = "appsclient"
	ClientTypeBatchClient       = "batchclient"
	ClientTypeBetaBatchClient   = "betabatchclient"
	ClientTypeAutoscalingClient = "autoscalingclient"
	ClientTypeStorageClient     = "storageclient"
)

// Mapping from resource kind to K8s apiserver API path. This is mostly pluralization, because
// K8s apiserver uses plural paths and this project singular.
// Must be kept in sync with the list of supported kinds.
var KindToAPIMapping = map[string]struct {
	// Kubernetes resource name.
	Resource string
	// Client type used by given resource, i.e. deployments are using extension client.
	ClientType ClientType
	// Is this object global scoped (not below a namespace).
	Namespaced bool
}{
	ResourceKindConfigMap:               {"configmaps", ClientTypeDefault, true},
	ResourceKindDaemonSet:               {"daemonsets", ClientTypeExtensionClient, true},
	ResourceKindDeployment:              {"deployments", ClientTypeExtensionClient, true},
	ResourceKindEvent:                   {"events", ClientTypeDefault, true},
	ResourceKindHorizontalPodAutoscaler: {"horizontalpodautoscalers", ClientTypeAutoscalingClient, true},
	ResourceKindIngress:                 {"ingresses", ClientTypeExtensionClient, true},
	ResourceKindJob:                     {"jobs", ClientTypeBatchClient, true},
	ResourceKindCronJob:                 {"cronjobs", ClientTypeBetaBatchClient, true},
	ResourceKindLimitRange:              {"limitrange", ClientTypeDefault, true},
	ResourceKindNamespace:               {"namespaces", ClientTypeDefault, false},
	ResourceKindNode:                    {"nodes", ClientTypeDefault, false},
	ResourceKindPersistentVolumeClaim:   {"persistentvolumeclaims", ClientTypeDefault, true},
	ResourceKindPersistentVolume:        {"persistentvolumes", ClientTypeDefault, false},
	ResourceKindPod:                     {"pods", ClientTypeDefault, true},
	ResourceKindReplicaSet:              {"replicasets", ClientTypeExtensionClient, true},
	ResourceKindReplicationController:   {"replicationcontrollers", ClientTypeDefault, true},
	ResourceKindResourceQuota:           {"resourcequotas", ClientTypeDefault, true},
	ResourceKindSecret:                  {"secrets", ClientTypeDefault, true},
	ResourceKindService:                 {"services", ClientTypeDefault, true},
	ResourceKindStatefulSet:             {"statefulsets", ClientTypeAppsClient, true},
	ResourceKindStorageClass:            {"storageclasses", ClientTypeStorageClient, false},
	ResourceKindEndpoint:                {"endpoints", ClientTypeDefault, true},
}

func GetResourceKinds() sets.String {
	kinds := sets.NewString()
	for kind := range KindToAPIMapping {
		kinds.Insert(kind)
	}
	return kinds
}
