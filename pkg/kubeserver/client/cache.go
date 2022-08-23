package client

import (
	"context"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	version2 "k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/kubernetes"
	apps "k8s.io/client-go/listers/apps/v1"
	autoscalingv1 "k8s.io/client-go/listers/autoscaling/v1"
	batch "k8s.io/client-go/listers/batch/v1"
	"k8s.io/client-go/listers/batch/v1beta1"
	v1 "k8s.io/client-go/listers/core/v1"
	networking "k8s.io/client-go/listers/networking/v1"
	rbac "k8s.io/client-go/listers/rbac/v1"
	storage "k8s.io/client-go/listers/storage/v1"
	"k8s.io/client-go/tools/cache"
	"time"
	api1 "yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/models/manager"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/pkg/errors"
)

var (
	eventWorkMan = appsrv.NewWorkerManager("K8SEventHandlerWorkerManager", 4, 10240, true)
)

type CacheFactory struct {
	versionInfo *version2.Info
	stopChan    chan struct{}
	// sharedInformerFactory  informers.SharedInformerFactory
	dynamicInformerFactory dynamicinformer.DynamicSharedInformerFactory
	bidirectionalSync      bool
}

func buildCacheController(
	cluster manager.ICluster,
	client *kubernetes.Clientset,
	dynamicClient dynamic.Interface,
	restMapper meta.PriorityRESTMapper,
) (*CacheFactory, error) {
	versionInfo, err := client.ServerVersion()
	if err != nil {
		log.Errorf("get k8s server version error")
		return nil, errors.Wrap(err, "Get K8s Server version")
	}

	stop := make(chan struct{})
	cacheF := &CacheFactory{
		stopChan:          stop,
		bidirectionalSync: false,
		versionInfo:       versionInfo,
	}

	// sharedInformerFactory := informers.NewSharedInformerFactory(client, defaultResyncPeriod)
	// sharedInformerFactory := informers.NewSharedInformerFactory(client, 0)
	// dynamicInformerFactory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, defaultResyncPeriod)
	dynamicInformerFactory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, 0)

	// Start all Resources defined in KindToResourceMap
	informerSyncs := make([]cache.InformerSynced, 0)
	// FIXME: Minor not bound to be a number, e.g. '15+'
	resourceMap := api.GetKindToResourceMap(versionInfo)

	// for _, value := range resourceMap {
	//	genericInformer, err := sharedInformerFactory.ForResource(value.GroupVersionResourceKind.GroupVersionResource)
	//	if err != nil {
	//		return nil, err
	//	}
	//	resMan := cluster.GetK8sResourceManager(value.GroupVersionResourceKind.Kind)
	//	if resMan != nil {
	//		// register informer event handler
	//		genericInformer.Informer().AddEventHandler(newEventHandler(cacheF, cluster, resMan))
	//		informerSyncs = append(informerSyncs, genericInformer.Informer().HasSynced)
	//	}
	//	// go genericInformer.Informer().Run(stop)
	//}

	// Start all dynamic rest mapper resource
	for _, res := range resourceMap {
		genericInformer := dynamicInformerFactory.ForResource(res.GroupVersionResourceKind.GroupVersionResource)
		informerSyncs = append(informerSyncs, genericInformer.Informer().HasSynced)

		resMan := cluster.GetK8sResourceManager(res.GroupVersionResourceKind.Kind)
		if resMan == nil {
			// FIXME: Event, Endpoints, HorizontalPodAutoscaler don't have resource manager
			log.Errorf("Resource manager NOT FOUND, Resource Name: %v", res.GroupVersionResourceKind.Kind)
			continue
		}

		genericInformer.Informer().AddEventHandler(newEventHandler(cacheF, cluster, resMan))
		// go genericInformer.Informer().Run(stop)
	}

	//sharedInformerFactory.Start(stop)
	dynamicInformerFactory.Start(stop)

	if !cache.WaitForCacheSync(stop, informerSyncs...) {
		return nil, errors.Errorf("informers not synced")
	}

	//cacheF.sharedInformerFactory = sharedInformerFactory
	cacheF.dynamicInformerFactory = dynamicInformerFactory
	return cacheF, nil
}

func (c *CacheFactory) PodLister() cache.GenericLister {
	return c.dynamicInformerFactory.ForResource(api.GetResourceMapByVersion(api1.ResourceNamePod, c.versionInfo).GroupVersionResourceKind.GroupVersionResource).Lister()
}

func (c *CacheFactory) EventLister() cache.GenericLister {
	return c.dynamicInformerFactory.ForResource(api.GetResourceMapByVersion(api1.ResourceNameEvent, c.versionInfo).GroupVersionResourceKind.GroupVersionResource).Lister()
}

func (c *CacheFactory) ConfigMapLister() v1.ConfigMapLister {
	return v1.NewConfigMapLister(c.dynamicInformerFactory.ForResource(api.GetResourceMapByVersion(api1.ResourceNameConfigMap, c.versionInfo).GroupVersionResourceKind.GroupVersionResource).Informer().GetIndexer())
}

func (c *CacheFactory) SecretLister() v1.SecretLister {
	return v1.NewSecretLister(c.dynamicInformerFactory.ForResource(api.GetResourceMapByVersion(api1.ResourceNameSecret, c.versionInfo).GroupVersionResourceKind.GroupVersionResource).Informer().GetIndexer())
}

func (c *CacheFactory) DeploymentLister() apps.DeploymentLister {
	return apps.NewDeploymentLister(c.dynamicInformerFactory.ForResource(api.GetResourceMapByVersion(api1.ResourceNameDeployment, c.versionInfo).GroupVersionResourceKind.GroupVersionResource).Informer().GetIndexer())
}

func (c *CacheFactory) DaemonSetLister() apps.DaemonSetLister {
	return apps.NewDaemonSetLister(c.dynamicInformerFactory.ForResource(api.GetResourceMapByVersion(api1.ResourceNameDaemonSet, c.versionInfo).GroupVersionResourceKind.GroupVersionResource).Informer().GetIndexer())
}

func (c *CacheFactory) StatefulSetLister() apps.StatefulSetLister {
	return apps.NewStatefulSetLister(c.dynamicInformerFactory.ForResource(api.GetResourceMapByVersion(api1.ResourceNameStatefulSet, c.versionInfo).GroupVersionResourceKind.GroupVersionResource).Informer().GetIndexer())
}

func (c *CacheFactory) NodeLister() cache.GenericLister {
	return c.dynamicInformerFactory.ForResource(api.GetResourceMapByVersion(api1.ResourceNameNode, c.versionInfo).GroupVersionResourceKind.GroupVersionResource).Lister()
}

func (c *CacheFactory) EndpointLister() v1.EndpointsLister {
	return v1.NewEndpointsLister(c.dynamicInformerFactory.ForResource(api.GetResourceMapByVersion(api1.ResourceNameEndpoint, c.versionInfo).GroupVersionResourceKind.GroupVersionResource).Informer().GetIndexer())
}

func (c *CacheFactory) HPALister() autoscalingv1.HorizontalPodAutoscalerLister {
	return autoscalingv1.NewHorizontalPodAutoscalerLister(c.dynamicInformerFactory.ForResource(api.GetResourceMapByVersion(api1.ResourceNameHorizontalPodAutoscaler, c.versionInfo).GroupVersionResourceKind.GroupVersionResource).Informer().GetIndexer())
}

func (c *CacheFactory) IngressLister() networking.IngressLister {
	return networking.NewIngressLister(c.dynamicInformerFactory.ForResource(api.GetResourceMapByVersion(api1.ResourceNameIngress, c.versionInfo).GroupVersionResourceKind.GroupVersionResource).Informer().GetIndexer())
}

func (c *CacheFactory) ServiceLister() v1.ServiceLister {
	return v1.NewServiceLister(c.dynamicInformerFactory.ForResource(api.GetResourceMapByVersion(api1.ResourceNameService, c.versionInfo).GroupVersionResourceKind.GroupVersionResource).Informer().GetIndexer())
}

func (c *CacheFactory) LimitRangeLister() v1.LimitRangeLister {
	return v1.NewLimitRangeLister(c.dynamicInformerFactory.ForResource(api.GetResourceMapByVersion(api1.ResourceNameLimitRange, c.versionInfo).GroupVersionResourceKind.GroupVersionResource).Informer().GetIndexer())
}

func (c *CacheFactory) NamespaceLister() v1.NamespaceLister {
	return v1.NewNamespaceLister(c.dynamicInformerFactory.ForResource(api.GetResourceMapByVersion(api1.ResourceNameNamespace, c.versionInfo).GroupVersionResourceKind.GroupVersionResource).Informer().GetIndexer())
}

func (c *CacheFactory) ReplicationControllerLister() v1.ReplicationControllerLister {
	return v1.NewReplicationControllerLister(c.dynamicInformerFactory.ForResource(api.GetResourceMapByVersion("replicationcontrollers", c.versionInfo).GroupVersionResourceKind.GroupVersionResource).Informer().GetIndexer())
}

func (c *CacheFactory) ReplicaSetLister() cache.GenericLister {
	return c.dynamicInformerFactory.ForResource(api.GetResourceMapByVersion(api1.ResourceNameReplicaSet, c.versionInfo).GroupVersionResourceKind.GroupVersionResource).Lister()
}

func (c *CacheFactory) JobLister() batch.JobLister {
	return batch.NewJobLister(c.dynamicInformerFactory.ForResource(api.GetResourceMapByVersion(api1.ResourceNameJob, c.versionInfo).GroupVersionResourceKind.GroupVersionResource).Informer().GetIndexer())
}

func (c *CacheFactory) CronJobLister() v1beta1.CronJobLister {
	return v1beta1.NewCronJobLister(c.dynamicInformerFactory.ForResource(api.GetResourceMapByVersion(api1.ResourceNameCronJob, c.versionInfo).GroupVersionResourceKind.GroupVersionResource).Informer().GetIndexer())
}

func (c *CacheFactory) PVLister() v1.PersistentVolumeLister {
	return v1.NewPersistentVolumeLister(c.dynamicInformerFactory.ForResource(api.GetResourceMapByVersion(api1.ResourceNamePersistentVolume, c.versionInfo).GroupVersionResourceKind.GroupVersionResource).Informer().GetIndexer())
}

func (c *CacheFactory) PVCLister() v1.PersistentVolumeClaimLister {
	return v1.NewPersistentVolumeClaimLister(c.dynamicInformerFactory.ForResource(api.GetResourceMapByVersion(api1.ResourceNamePersistentVolumeClaim, c.versionInfo).GroupVersionResourceKind.GroupVersionResource).Informer().GetIndexer())
}

func (c *CacheFactory) StorageClassLister() storage.StorageClassLister {
	return storage.NewStorageClassLister(c.dynamicInformerFactory.ForResource(api.GetResourceMapByVersion(api1.ResourceNameStorageClass, c.versionInfo).GroupVersionResourceKind.GroupVersionResource).Informer().GetIndexer())
}

func (c *CacheFactory) ResourceQuotaLister() v1.ResourceQuotaLister {
	return v1.NewResourceQuotaLister(c.dynamicInformerFactory.ForResource(api.GetResourceMapByVersion(api1.ResourceNameResourceQuota, c.versionInfo).GroupVersionResourceKind.GroupVersionResource).Informer().GetIndexer())
}

func (c *CacheFactory) RoleLister() rbac.RoleLister {
	return rbac.NewRoleLister(c.dynamicInformerFactory.ForResource(api.GetResourceMapByVersion(api1.ResourceNameRole, c.versionInfo).GroupVersionResourceKind.GroupVersionResource).Informer().GetIndexer())
}

func (c *CacheFactory) ClusterRoleLister() rbac.ClusterRoleLister {
	return rbac.NewClusterRoleLister(c.dynamicInformerFactory.ForResource(api.GetResourceMapByVersion(api1.ResourceNameClusterRole, c.versionInfo).GroupVersionResourceKind.GroupVersionResource).Informer().GetIndexer())
}

func (c *CacheFactory) RoleBindingLister() rbac.RoleBindingLister {
	return rbac.NewRoleBindingLister(c.dynamicInformerFactory.ForResource(api.GetResourceMapByVersion(api1.ResourceNameRoleBinding, c.versionInfo).GroupVersionResourceKind.GroupVersionResource).Informer().GetIndexer())
}

func (c *CacheFactory) ClusterRoleBindingLister() rbac.ClusterRoleBindingLister {
	return rbac.NewClusterRoleBindingLister(c.dynamicInformerFactory.ForResource(api.GetResourceMapByVersion(api1.ResourceNameClusterRoleBinding, c.versionInfo).GroupVersionResourceKind.GroupVersionResource).Informer().GetIndexer())
}

func (c *CacheFactory) ServiceAccountLister() v1.ServiceAccountLister {
	return v1.NewServiceAccountLister(c.dynamicInformerFactory.ForResource(api.GetResourceMapByVersion(api1.ResourceNameServiceAccount, c.versionInfo).GroupVersionResourceKind.GroupVersionResource).Informer().GetIndexer())
}

func (c *CacheFactory) EnableBidirectionalSync() {
	c.bidirectionalSync = true
}

func (c *CacheFactory) DisableBidirectionalSync() {
	c.bidirectionalSync = false
}

type eventHandler struct {
	cacheFactory *CacheFactory
	cluster      manager.ICluster
	manager      manager.IK8sResourceManager
}

func newEventHandler(cacheF *CacheFactory, cluster manager.ICluster, man manager.IK8sResourceManager) cache.ResourceEventHandler {
	return &eventHandler{
		cacheFactory: cacheF,
		cluster:      cluster,
		manager:      man,
	}
}

func (h eventHandler) run(f func(ctx context.Context, userCred mcclient.TokenCredential, cls manager.ICluster)) {
	// cacheFactory must enable bidirectional sync
	if !h.cacheFactory.bidirectionalSync {
		return
	}

	adminCred := auth.AdminCredential()
	ctx := context.Background()
	now := time.Now()
	ms := now.UnixMilli()
	ctx = context.WithValue(ctx, "Time", ms)
	lockman.LockClass(ctx, h.manager, db.GetLockClassKey(h.manager, adminCred))
	defer lockman.ReleaseClass(ctx, h.manager, db.GetLockClassKey(h.manager, adminCred))

	// eventWorkMan.Run(func() {
	// f(ctx, adminCred, h.cluster)
	// }, nil, nil)
	f(ctx, adminCred, h.cluster)
}

func (h eventHandler) OnAdd(obj interface{}) {
	h.run(func(ctx context.Context, userCred mcclient.TokenCredential, cls manager.ICluster) {
		h.manager.OnRemoteObjectCreate(ctx, userCred, cls, h.manager, obj.(runtime.Object))
	})
}

func (h eventHandler) OnUpdate(oldObj, newObj interface{}) {
	h.run(func(ctx context.Context, userCred mcclient.TokenCredential, cls manager.ICluster) {
		h.manager.OnRemoteObjectUpdate(ctx, userCred, cls, h.manager, oldObj.(runtime.Object), newObj.(runtime.Object))
	})
}

func (h eventHandler) OnDelete(obj interface{}) {
	h.run(func(ctx context.Context, userCred mcclient.TokenCredential, cls manager.ICluster) {
		h.manager.OnRemoteObjectDelete(ctx, userCred, cls, h.manager, obj.(runtime.Object))
	})
}
