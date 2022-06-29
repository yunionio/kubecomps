package client

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	apps "k8s.io/client-go/listers/apps/v1"
	autoscalingv1 "k8s.io/client-go/listers/autoscaling/v1"
	batch "k8s.io/client-go/listers/batch/v1"
	batch2 "k8s.io/client-go/listers/batch/v1beta1"
	"k8s.io/client-go/listers/core/v1"
	extensions "k8s.io/client-go/listers/extensions/v1beta1"
	rbac "k8s.io/client-go/listers/rbac/v1"
	storage "k8s.io/client-go/listers/storage/v1"
	"k8s.io/client-go/tools/cache"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/pkg/errors"

	// kapi "yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/models/manager"
)

var (
	eventWorkMan = appsrv.NewWorkerManager("K8SEventHandlerWorkerManager", 4, 10240, true)
)

type CacheFactory struct {
	stopChan               chan struct{}
	sharedInformerFactory  informers.SharedInformerFactory
	dynamicInformerFactory dynamicinformer.DynamicSharedInformerFactory
	bidirectionalSync      bool
}

func buildCacheController(
	cluster manager.ICluster,
	client *kubernetes.Clientset,
	dynamicClient dynamic.Interface,
	restMapper meta.PriorityRESTMapper,
) (*CacheFactory, error) {
	stop := make(chan struct{})
	cacheF := &CacheFactory{
		stopChan:          stop,
		bidirectionalSync: false,
	}
	// sharedInformerFactory := informers.NewSharedInformerFactory(client, defaultResyncPeriod)
	sharedInformerFactory := informers.NewSharedInformerFactory(client, 0)
	// dynamicInformerFactory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, defaultResyncPeriod)
	// dynamicInformerFactory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, 0)

	// Start all Resources defined in KindToResourceMap
	informerSyncs := make([]cache.InformerSynced, 0)
	for _, value := range api.KindToResourceMap {
		genericInformer, err := sharedInformerFactory.ForResource(value.GroupVersionResourceKind.GroupVersionResource)
		if err != nil {
			return nil, err
		}
		resMan := cluster.GetK8sResourceManager(value.GroupVersionResourceKind.Kind)
		if resMan != nil {
			// register informer event handler
			genericInformer.Informer().AddEventHandler(newEventHandler(cacheF, cluster, resMan))
			informerSyncs = append(informerSyncs, genericInformer.Informer().HasSynced)
		}
		// go genericInformer.Informer().Run(stop)
	}

	// Start all dynamic rest mapper resource
	/*for _, res := range restMapper.ResourcePriority {
		genericInformer := dynamicInformerFactory.ForResource(res)
		informerSyncs = append(informerSyncs, genericInformer.Informer().HasSynced)
		genericInformer.Informer().AddEventHandler(
			cache.ResourceEventHandlerFuncs{
				AddFunc:    nil,
				UpdateFunc: nil,
				DeleteFunc: nil,
			})
		// go genericInformer.Informer().Run(stop)
	}*/

	sharedInformerFactory.Start(stop)
	//dynamicInformerFactory.Start(stop)

	if !cache.WaitForCacheSync(stop, informerSyncs...) {
		return nil, errors.Errorf("informers not synced")
	}
	cacheF.sharedInformerFactory = sharedInformerFactory
	return cacheF, nil
}

func (c *CacheFactory) PodLister() v1.PodLister {
	return c.sharedInformerFactory.Core().V1().Pods().Lister()
}

func (c *CacheFactory) EventLister() v1.EventLister {
	return c.sharedInformerFactory.Core().V1().Events().Lister()
}

func (c *CacheFactory) ConfigMapLister() v1.ConfigMapLister {
	return c.sharedInformerFactory.Core().V1().ConfigMaps().Lister()
}

func (c *CacheFactory) SecretLister() v1.SecretLister {
	return c.sharedInformerFactory.Core().V1().Secrets().Lister()
}

func (c *CacheFactory) DeploymentLister() apps.DeploymentLister {
	return c.sharedInformerFactory.Apps().V1().Deployments().Lister()
}

func (c *CacheFactory) DaemonSetLister() apps.DaemonSetLister {
	return c.sharedInformerFactory.Apps().V1().DaemonSets().Lister()
}

func (c *CacheFactory) StatefulSetLister() apps.StatefulSetLister {
	return c.sharedInformerFactory.Apps().V1().StatefulSets().Lister()
}

func (c *CacheFactory) NodeLister() v1.NodeLister {
	return c.sharedInformerFactory.Core().V1().Nodes().Lister()
}

func (c *CacheFactory) EndpointLister() v1.EndpointsLister {
	return c.sharedInformerFactory.Core().V1().Endpoints().Lister()
}

func (c *CacheFactory) HPALister() autoscalingv1.HorizontalPodAutoscalerLister {
	return c.sharedInformerFactory.Autoscaling().V1().HorizontalPodAutoscalers().Lister()
}

func (c *CacheFactory) IngressLister() extensions.IngressLister {
	return c.sharedInformerFactory.Extensions().V1beta1().Ingresses().Lister()
}

func (c *CacheFactory) ServiceLister() v1.ServiceLister {
	return c.sharedInformerFactory.Core().V1().Services().Lister()
}

func (c *CacheFactory) LimitRangeLister() v1.LimitRangeLister {
	return c.sharedInformerFactory.Core().V1().LimitRanges().Lister()
}

func (c *CacheFactory) NamespaceLister() v1.NamespaceLister {
	return c.sharedInformerFactory.Core().V1().Namespaces().Lister()
}

func (c *CacheFactory) ReplicationControllerLister() v1.ReplicationControllerLister {
	return c.sharedInformerFactory.Core().V1().ReplicationControllers().Lister()
}

func (c *CacheFactory) ReplicaSetLister() apps.ReplicaSetLister {
	return c.sharedInformerFactory.Apps().V1().ReplicaSets().Lister()
}

func (c *CacheFactory) JobLister() batch.JobLister {
	return c.sharedInformerFactory.Batch().V1().Jobs().Lister()
}

func (c *CacheFactory) CronJobLister() batch2.CronJobLister {
	return c.sharedInformerFactory.Batch().V1beta1().CronJobs().Lister()
}

func (c *CacheFactory) PVLister() v1.PersistentVolumeLister {
	return c.sharedInformerFactory.Core().V1().PersistentVolumes().Lister()
}

func (c *CacheFactory) PVCLister() v1.PersistentVolumeClaimLister {
	return c.sharedInformerFactory.Core().V1().PersistentVolumeClaims().Lister()
}

func (c *CacheFactory) StorageClassLister() storage.StorageClassLister {
	return c.sharedInformerFactory.Storage().V1().StorageClasses().Lister()
}

func (c *CacheFactory) ResourceQuotaLister() v1.ResourceQuotaLister {
	return c.sharedInformerFactory.Core().V1().ResourceQuotas().Lister()
}

func (c *CacheFactory) RoleLister() rbac.RoleLister {
	return c.sharedInformerFactory.Rbac().V1().Roles().Lister()
}

func (c *CacheFactory) ClusterRoleLister() rbac.ClusterRoleLister {
	return c.sharedInformerFactory.Rbac().V1().ClusterRoles().Lister()
}

func (c *CacheFactory) RoleBindingLister() rbac.RoleBindingLister {
	return c.sharedInformerFactory.Rbac().V1().RoleBindings().Lister()
}

func (c *CacheFactory) ClusterRoleBindingLister() rbac.ClusterRoleBindingLister {
	return c.sharedInformerFactory.Rbac().V1().ClusterRoleBindings().Lister()
}

func (c *CacheFactory) ServiceAccountLister() v1.ServiceAccountLister {
	return c.sharedInformerFactory.Core().V1().ServiceAccounts().Lister()
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
