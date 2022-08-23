package common

import (
	apps "k8s.io/api/apps/v1"
	autoscaling "k8s.io/api/autoscaling/v1"
	batch "k8s.io/api/batch/v1"
	batch2 "k8s.io/api/batch/v1beta1"
	"k8s.io/api/core/v1"
	v12 "k8s.io/api/networking/v1"
	rbac "k8s.io/api/rbac/v1"
	storage "k8s.io/api/storage/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
	api "yunion.io/x/kubecomps/pkg/kubeserver/types/apis"
)

// ResourceChannels struct holds channels to resource lists. Each list channel is paired with
// an error channel which *must* be read sequentially: first read the list channel and then the error channel.
type ResourceChannels struct {
	// List and error channels to Replication Controllers.
	ReplicationControllerList ReplicationControllerListChannel

	// List and error channels to Replica Sets
	ReplicaSetList ReplicaSetListChannel

	// List and error channels to Deployments
	DeploymentList DeploymentListChannel

	// List and error channels to Daemon Sets
	DaemonSetList DaemonSetListChannel

	// List and error channels to Jobs.
	JobList JobListChannel

	// List and error channels to Cron Jobs.
	CronJobList CronJobListChannel

	// List and error channels to Services
	ServiceList ServiceListChannel

	// List and error channels to Endpoints.
	EndpointList EndpointListChannel

	// List and error channels to Ingresses.
	IngressList IngressListChannel

	// List and error channels to Pods
	PodList PodListChannel

	// List and error channels to Events.
	EventList EventListChannel

	// List and error channels to LimitRanges.
	LimitRangeList LimitRangeListChannel

	// List and error channels to Nodes.
	NodeList NodeListChannel

	// List and error channels to Namespaces.
	NamespaceList NamespaceListChannel

	// List and error channels to StatefulSets.
	StatefulSetList StatefulSetListChannel

	// List and error channels to ConfigMaps.
	ConfigMapList ConfigMapListChannel

	// List and error channels to Secrets.
	SecretList SecretListChannel

	// List and error channels to PersistentVolumes
	PersistentVolumeList PersistentVolumeListChannel

	// List and error channels to PersistentVolumeClaims
	PersistentVolumeClaimList PersistentVolumeClaimListChannel

	// List and error channels to ResourceQuotas
	ResourceQuotaList ResourceQuotaListChannel

	// List and error channels to HorizontalPodAutoscalers
	HorizontalPodAutoscalerList HorizontalPodAutoscalerListChannel

	// List and error channels to StorageClasses
	StorageClassList StorageClassListChannel

	// List and error channels to Roles
	RoleList RoleListChannel

	// List and error channels to ClusterRoles
	ClusterRoleList ClusterRoleListChannel

	// List and error channels to RoleBindings
	RoleBindingList RoleBindingListChannel

	// List and error channels to ClusterRoleBindings
	ClusterRoleBindingList ClusterRoleBindingListChannel
}

// IngressListChannel is a list and error channels to Ingresss.
type IngressListChannel struct {
	List  chan []*v12.Ingress
	Error chan error
}

func GetIngressListChannel(indexer *client.CacheFactory, nsQuery *NamespaceQuery) IngressListChannel {
	return GetIngressListChannelWithOptions(indexer, nsQuery, api.ListEverything)
}

func GetIngressListChannelWithOptions(indexer *client.CacheFactory, nsQuery *NamespaceQuery, options metaV1.ListOptions) IngressListChannel {
	channel := IngressListChannel{
		List:  make(chan []*v12.Ingress),
		Error: make(chan error),
	}
	go func() {
		list, err := indexer.IngressLister().Ingresses(nsQuery.ToRequestParam()).List(labels.Everything())
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

type ServiceListChannel struct {
	List  chan []*v1.Service
	Error chan error
}

func GetServiceListChannel(indexer *client.CacheFactory, nsQuery *NamespaceQuery) ServiceListChannel {
	return GetServiceListChannelWithOptions(indexer, nsQuery, labels.Everything())
}

func GetServiceListChannelWithOptions(indexer *client.CacheFactory, nsQuery *NamespaceQuery, options labels.Selector) ServiceListChannel {
	channel := ServiceListChannel{
		List:  make(chan []*v1.Service),
		Error: make(chan error),
	}
	go func() {
		list, err := indexer.ServiceLister().Services(nsQuery.ToRequestParam()).List(options)
		channel.List <- list
		channel.Error <- err
	}()
	return channel
}

type LimitRangeListChannel struct {
	List  chan []*v1.LimitRange
	Error chan error
}

func GetLimitRangeListChannel(indexer *client.CacheFactory, nsQuery *NamespaceQuery) LimitRangeListChannel {
	channel := LimitRangeListChannel{
		List:  make(chan []*v1.LimitRange),
		Error: make(chan error),
	}

	go func() {
		list, err := indexer.LimitRangeLister().LimitRanges(nsQuery.ToRequestParam()).List(labels.Everything())
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

type NodeListChannel struct {
	List  chan []*v1.Node
	Error chan error
}

func GetNodeListChannel(indexer *client.CacheFactory) NodeListChannel {
	channel := NodeListChannel{
		List:  make(chan []*v1.Node),
		Error: make(chan error),
	}

	go func() {
		list, err := indexer.NodeLister().List(labels.Everything())
		res := make([]*v1.Node, len(list))
		for idx, l := range list {
			newObj := &v1.Node{}
			if err = runtime.DefaultUnstructuredConverter.FromUnstructured(l.(*unstructured.Unstructured).Object, newObj); err != nil {
				return
			}
			res[idx] = newObj
		}
		channel.List <- res
		channel.Error <- err
	}()

	return channel
}

type NamespaceListChannel struct {
	List  chan []*v1.Namespace
	Error chan error
}

func GetNamespaceListChannel(indexer *client.CacheFactory) NamespaceListChannel {
	channel := NamespaceListChannel{
		List:  make(chan []*v1.Namespace),
		Error: make(chan error),
	}

	go func() {
		list, err := indexer.NamespaceLister().List(labels.Everything())
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

type EventListChannel struct {
	List  chan []*v1.Event
	Error chan error
}

func GetEventListChannel(indexer *client.CacheFactory, nsQuery *NamespaceQuery) EventListChannel {
	return GetEventListChannelWithOptions(indexer, nsQuery, labels.Everything())
}

func GetEventListChannelWithOptions(indexer *client.CacheFactory,
	nsQuery *NamespaceQuery, options labels.Selector) EventListChannel {
	channel := EventListChannel{
		List:  make(chan []*v1.Event),
		Error: make(chan error),
	}

	go func() {
		list, err := indexer.EventLister().ByNamespace(nsQuery.ToRequestParam()).List(options)
		res := make([]*v1.Event, len(list))
		for idx, l := range list {
			newObj := &v1.Event{}
			if err = runtime.DefaultUnstructuredConverter.FromUnstructured(l.(*unstructured.Unstructured).Object, newObj); err != nil {
				return
			}
			res[idx] = newObj
		}
		channel.List <- res
		channel.Error <- err
	}()

	return channel
}

type EndpointListChannel struct {
	List  chan []*v1.Endpoints
	Error chan error
}

func GetEndpointListChannel(indexer *client.CacheFactory, nsQuery *NamespaceQuery) EndpointListChannel {
	return GetEndpointListChannelWithOptions(indexer, nsQuery, labels.Everything())
}

func GetEndpointListChannelWithOptions(indexer *client.CacheFactory,
	nsQuery *NamespaceQuery, opt labels.Selector) EndpointListChannel {
	channel := EndpointListChannel{
		List:  make(chan []*v1.Endpoints),
		Error: make(chan error),
	}

	go func() {
		list, err := indexer.EndpointLister().Endpoints(nsQuery.ToRequestParam()).List(opt)

		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

// PodListChannel is a list and error channels to Nodes
type PodListChannel struct {
	List  chan []*v1.Pod
	Error chan error
}

func GetPodListChannel(indexer *client.CacheFactory, nsQuery *NamespaceQuery) PodListChannel {
	return GetPodListChannelWithOptions(indexer, nsQuery, labels.Everything())
}

func GetPodListChannelWithOptions(indexer *client.CacheFactory, nsQuery *NamespaceQuery, options labels.Selector) PodListChannel {
	channel := PodListChannel{
		List:  make(chan []*v1.Pod),
		Error: make(chan error),
	}

	go func() {
		list, err := indexer.PodLister().ByNamespace(nsQuery.ToRequestParam()).List(options)
		res := make([]*v1.Pod, len(list))
		for idx, l := range list {
			newObj := &v1.Pod{}
			if err = runtime.DefaultUnstructuredConverter.FromUnstructured(l.(*unstructured.Unstructured).Object, newObj); err != nil {
				return
			}
			res[idx] = newObj
		}
		channel.List <- res
		channel.Error <- err
	}()

	return channel
}

type ReplicationControllerListChannel struct {
	List  chan []*v1.ReplicationController
	Error chan error
}

func GetReplicationControllerListChannel(indexer *client.CacheFactory,
	nsQuery *NamespaceQuery) ReplicationControllerListChannel {

	channel := ReplicationControllerListChannel{
		List:  make(chan []*v1.ReplicationController),
		Error: make(chan error),
	}

	go func() {
		list, err := indexer.ReplicationControllerLister().ReplicationControllers(nsQuery.ToRequestParam()).
			List(labels.Everything())
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

type DeploymentListChannel struct {
	List  chan []*apps.Deployment
	Error chan error
}

func GetDeploymentListChannel(indexer *client.CacheFactory,
	nsQuery *NamespaceQuery) DeploymentListChannel {
	channel := DeploymentListChannel{
		List:  make(chan []*apps.Deployment),
		Error: make(chan error),
	}
	go func() {
		list, err := indexer.DeploymentLister().Deployments(nsQuery.ToRequestParam()).List(labels.Everything())
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

type ReplicaSetListChannel struct {
	List  chan []*apps.ReplicaSet
	Error chan error
}

func GetReplicaSetListChannel(indexer *client.CacheFactory,
	nsQuery *NamespaceQuery) ReplicaSetListChannel {
	return GetReplicaSetListChannelWithOptions(indexer, nsQuery, labels.Everything())
}

func GetReplicaSetListChannelWithOptions(indexer *client.CacheFactory, nsQuery *NamespaceQuery,
	options labels.Selector) ReplicaSetListChannel {
	channel := ReplicaSetListChannel{
		List:  make(chan []*apps.ReplicaSet),
		Error: make(chan error),
	}

	go func() {
		list, err := indexer.ReplicaSetLister().ByNamespace(nsQuery.ToRequestParam()).List(options)
		res := make([]*apps.ReplicaSet, len(list))
		for idx, l := range list {
			newObj := &apps.ReplicaSet{}
			if err = runtime.DefaultUnstructuredConverter.FromUnstructured(l.(*unstructured.Unstructured).Object, newObj); err != nil {
				return
			}
			res[idx] = newObj
		}
		channel.List <- res
		channel.Error <- err
	}()

	return channel
}

type DaemonSetListChannel struct {
	//List  chan []*apps.DaemonSet
	List  chan []*apps.DaemonSet
	Error chan error
}

func GetDaemonSetListChannel(indexer *client.CacheFactory, nsQuery *NamespaceQuery) DaemonSetListChannel {
	channel := DaemonSetListChannel{
		List:  make(chan []*apps.DaemonSet),
		Error: make(chan error),
	}

	go func() {
		list, err := indexer.DaemonSetLister().DaemonSets(nsQuery.ToRequestParam()).List(labels.Everything())
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

type JobListChannel struct {
	List  chan []*batch.Job
	Error chan error
}

func GetJobListChannel(indexer *client.CacheFactory,
	nsQuery *NamespaceQuery) JobListChannel {
	channel := JobListChannel{
		List:  make(chan []*batch.Job),
		Error: make(chan error),
	}

	go func() {
		list, err := indexer.JobLister().Jobs(nsQuery.ToRequestParam()).List(labels.Everything())
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

type CronJobListChannel struct {
	List  chan []*batch2.CronJob
	Error chan error
}

func GetCronJobListChannel(indexer *client.CacheFactory, nsQuery *NamespaceQuery) CronJobListChannel {
	channel := CronJobListChannel{
		List:  make(chan []*batch2.CronJob),
		Error: make(chan error),
	}

	go func() {
		list, err := indexer.CronJobLister().CronJobs(nsQuery.ToRequestParam()).List(labels.Everything())
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

// StatefulSetListChannel is a list and error channels to Nodes.
type StatefulSetListChannel struct {
	List  chan []*apps.StatefulSet
	Error chan error
}

func GetStatefulSetListChannel(indexer *client.CacheFactory, nsQuery *NamespaceQuery) StatefulSetListChannel {
	return GetStatefulSetListChannelWithOptions(indexer, nsQuery, labels.Everything())
}

func GetStatefulSetListChannelWithOptions(indexer *client.CacheFactory, nsQuery *NamespaceQuery, options labels.Selector) StatefulSetListChannel {
	channel := StatefulSetListChannel{
		List:  make(chan []*apps.StatefulSet),
		Error: make(chan error),
	}

	go func() {
		statefulSets, err := indexer.StatefulSetLister().StatefulSets(nsQuery.ToRequestParam()).List(options)
		channel.List <- statefulSets
		channel.Error <- err
	}()

	return channel
}

// ConfigMapListChannel is a list and error channels to ConfigMaps.
type ConfigMapListChannel struct {
	List  chan []*v1.ConfigMap
	Error chan error
}

func GetConfigMapListChannel(indexer *client.CacheFactory, nsQuery *NamespaceQuery) ConfigMapListChannel {
	return GetConfigMapListChannelWithOptions(indexer, nsQuery, labels.Everything())
}

func GetConfigMapListChannelWithOptions(indexer *client.CacheFactory, nsQuery *NamespaceQuery, options labels.Selector) ConfigMapListChannel {
	channel := ConfigMapListChannel{
		List:  make(chan []*v1.ConfigMap),
		Error: make(chan error),
	}

	go func() {
		list, err := indexer.ConfigMapLister().ConfigMaps(nsQuery.ToRequestParam()).List(options)
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

// SecretListChannel is a list and error channels to Secrets.
type SecretListChannel struct {
	List  chan []*v1.Secret
	Error chan error
}

func GetSecretListChannel(indexer *client.CacheFactory, nsQuery *NamespaceQuery) SecretListChannel {

	channel := SecretListChannel{
		List:  make(chan []*v1.Secret),
		Error: make(chan error),
	}

	go func() {
		list, err := indexer.SecretLister().Secrets(nsQuery.ToRequestParam()).List(labels.Everything())
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

// RoleListChannel is a list and error channels to Roles.
type RoleListChannel struct {
	List  chan []*rbac.Role
	Error chan error
}

func GetRoleListChannel(indexer *client.CacheFactory) RoleListChannel {
	channel := RoleListChannel{
		List:  make(chan []*rbac.Role),
		Error: make(chan error),
	}

	go func() {
		list, err := indexer.RoleLister().Roles("").List(labels.Everything())
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

// ClusterRoleListChannel is a list and error channels to ClusterRoles.
type ClusterRoleListChannel struct {
	List  chan []*rbac.ClusterRole
	Error chan error
}

func GetClusterRoleListChannel(indexer *client.CacheFactory) ClusterRoleListChannel {
	channel := ClusterRoleListChannel{
		List:  make(chan []*rbac.ClusterRole),
		Error: make(chan error),
	}

	go func() {
		list, err := indexer.ClusterRoleLister().List(labels.Everything())
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

// RoleBindingListChannel is a list and error channels to RoleBindings.
type RoleBindingListChannel struct {
	List  chan []*rbac.RoleBinding
	Error chan error
}

func GetRoleBindingListChannel(indexer *client.CacheFactory) RoleBindingListChannel {
	channel := RoleBindingListChannel{
		List:  make(chan []*rbac.RoleBinding),
		Error: make(chan error),
	}

	go func() {
		list, err := indexer.RoleBindingLister().RoleBindings("").List(labels.Everything())
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

// ClusterRoleBindingListChannel is a list and error channels to ClusterRoleBindings.
type ClusterRoleBindingListChannel struct {
	List  chan []*rbac.ClusterRoleBinding
	Error chan error
}

func GetClusterRoleBindingListChannel(indexer *client.CacheFactory) ClusterRoleBindingListChannel {
	channel := ClusterRoleBindingListChannel{
		List:  make(chan []*rbac.ClusterRoleBinding),
		Error: make(chan error),
	}

	go func() {
		list, err := indexer.ClusterRoleBindingLister().List(labels.Everything())
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

// PersistentVolumeListChannel is a list and error channels to PersistentVolumes.
type PersistentVolumeListChannel struct {
	List  chan []*v1.PersistentVolume
	Error chan error
}

func GetPersistentVolumeListChannel(indexer *client.CacheFactory) PersistentVolumeListChannel {
	channel := PersistentVolumeListChannel{
		List:  make(chan []*v1.PersistentVolume),
		Error: make(chan error),
	}

	go func() {
		list, err := indexer.PVLister().List(labels.Everything())
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

// PersistentVolumeClaimListChannel is a list and error channels to PersistentVolumeClaims.
type PersistentVolumeClaimListChannel struct {
	List  chan []*v1.PersistentVolumeClaim
	Error chan error
}

func GetPersistentVolumeClaimListChannel(indexer *client.CacheFactory, nsQuery *NamespaceQuery) PersistentVolumeClaimListChannel {

	channel := PersistentVolumeClaimListChannel{
		List:  make(chan []*v1.PersistentVolumeClaim),
		Error: make(chan error),
	}

	go func() {
		list, err := indexer.PVCLister().PersistentVolumeClaims(nsQuery.ToRequestParam()).List(labels.Everything())
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

// ResourceQuotaListChannel is a list and error channels to ResourceQuotas.
type ResourceQuotaListChannel struct {
	List  chan []*v1.ResourceQuota
	Error chan error
}

func GetResourceQuotaListChannel(indexer *client.CacheFactory, nsQuery *NamespaceQuery) ResourceQuotaListChannel {
	channel := ResourceQuotaListChannel{
		List:  make(chan []*v1.ResourceQuota),
		Error: make(chan error),
	}

	go func() {
		list, err := indexer.ResourceQuotaLister().ResourceQuotas(nsQuery.ToRequestParam()).List(labels.Everything())
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

// HorizontalPodAutoscalerListChannel is a list and error channels.
type HorizontalPodAutoscalerListChannel struct {
	List  chan []*autoscaling.HorizontalPodAutoscaler
	Error chan error
}

func GetHorizontalPodAutoscalerListChannel(indexer *client.CacheFactory, nsQuery *NamespaceQuery) HorizontalPodAutoscalerListChannel {
	channel := HorizontalPodAutoscalerListChannel{
		List:  make(chan []*autoscaling.HorizontalPodAutoscaler),
		Error: make(chan error),
	}

	go func() {
		list, err := indexer.HPALister().HorizontalPodAutoscalers(nsQuery.ToRequestParam()).
			List(labels.Everything())
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

// StorageClassListChannel is a list and error channels to storage classes.
type StorageClassListChannel struct {
	List  chan []*storage.StorageClass
	Error chan error
}

func GetStorageClassListChannel(indexer *client.CacheFactory) StorageClassListChannel {
	channel := StorageClassListChannel{
		List:  make(chan []*storage.StorageClass),
		Error: make(chan error),
	}

	go func() {
		list, err := indexer.StorageClassLister().List(labels.Everything())
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}
