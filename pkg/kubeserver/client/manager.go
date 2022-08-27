package client

import (
	"sync"

	// "k8s.io/apimachinery/pkg/api/meta"
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	capi "yunion.io/x/kubecomps/pkg/kubeserver/client/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/clientv2"
	"yunion.io/x/kubecomps/pkg/kubeserver/models/manager"
)

const (
	ErrClusterAlreadyAdded = errors.Error("Cluster already added")
	ErrClusterNotRunning   = errors.Error("Cluster not running")
)

var (
	clustersManager *ClustersManager
)

type IClustersManager interface {
	AddClient(dbCluster manager.ICluster) error
	UpdateClient(dbCluster manager.ICluster) error
	RemoveClient(clusterId string) error
}

type ClustersManager struct {
	dbManager      manager.IClusterManager
	clusterManager map[string]*ClusterManager
	managerLock    sync.Mutex
	actionLock     sync.Mutex
}

func GetClustersManager() IClustersManager {
	return clustersManager
}

func InitClustersManager(dbManager manager.IClusterManager) error {
	runningClusters, err := dbManager.GetRunningClusters()
	if err != nil {
		return errors.Wrap(err, "get running clusters when init client clusters manager")
	}
	clustersManager = &ClustersManager{
		dbManager:      dbManager,
		clusterManager: make(map[string]*ClusterManager, 0),
		managerLock:    sync.Mutex{},
		actionLock:     sync.Mutex{},
	}
	clustersManager.initRunningClustersOnce(runningClusters)
	return nil
}

func (m *ClustersManager) initRunningClustersOnce(runningClusters []manager.ICluster) {
	succCnt := 0
	failedCnt := 0
	for i := 0; i < len(runningClusters); i++ {
		cluster := runningClusters[i]
		if err := m.AddClient(cluster); err != nil {
			log.Warningf("Add cluster %s client error: %v", cluster.GetName(), err)
			failedCnt++
			continue
		}
		succCnt++
	}
	log.Infof("init running clusters finished, success count %d, failed count %d", succCnt, failedCnt)
}

func (m *ClustersManager) getManager(clusterId string) *ClusterManager {
	m.managerLock.Lock()
	defer m.managerLock.Unlock()

	cm, ok := m.clusterManager[clusterId]
	if !ok {
		return nil
	}
	return cm
}

func (m *ClustersManager) addManager(man *ClusterManager) {
	m.managerLock.Lock()
	defer m.managerLock.Unlock()

	m.clusterManager[man.GetId()] = man
}

func (m *ClustersManager) AddClient(dbCluster manager.ICluster) error {
	m.actionLock.Lock()
	defer m.actionLock.Unlock()

	if status := dbCluster.GetStatus(); status != api.ClusterStatusRunning {
		return errors.Wrapf(ErrClusterNotRunning, "clusterId %s current status %s", dbCluster.GetName(), status)
	}

	clusterId := dbCluster.GetId()
	if cm := m.getManager(clusterId); cm != nil {
		return errors.Wrapf(ErrClusterAlreadyAdded, "clusterId %s", clusterId)
	}
	cm, err := m.buildManager(dbCluster)
	if err != nil {
		return errors.Wrap(err, "build cluster manager")
	}
	m.addManager(cm)

	return nil
}

func (m *ClustersManager) UpdateClient(dbCluster manager.ICluster) error {
	m.RemoveClient(dbCluster.GetId())
	return m.AddClient(dbCluster)
}

func (m *ClustersManager) RemoveClient(clusterId string) error {
	m.actionLock.Lock()
	defer m.actionLock.Unlock()

	cm := m.getManager(clusterId)
	if cm == nil {
		return nil
	}
	delete(m.clusterManager, clusterId)
	cm.Close()
	return nil
}

func (m *ClustersManager) parseAPIResources(dc discovery.DiscoveryInterface) ([]capi.ResourceMap, error) {
	lists, err := dc.ServerPreferredResources()
	if err != nil {
		return nil, errors.Wrapf(err, "ServerPreferredResources for discovery client")
	}
	resources := []capi.ResourceMap{}
	for _, list := range lists {
		if len(list.APIResources) == 0 {
			continue
		}
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			log.Errorf("ParseGroupVersion %q", list.GroupVersion)
			continue
		}
		for _, resource := range list.APIResources {
			if len(resource.Verbs) == 0 {
				continue
			}
			res := capi.ResourceMap{
				Namespaced: resource.Namespaced,
				GroupVersionResourceKind: capi.GroupVersionResourceKind{
					GroupVersionResource: schema.GroupVersionResource{
						Group:    gv.Group,
						Version:  gv.Version,
						Resource: resource.Name,
					},
					Kind: resource.Kind,
				},
			}
			ver, _ := dc.ServerVersion()
			if res.GroupVersionResourceKind.Kind == api.KindNameCronJob {
				log.Errorf("-=========CRONJOB %v, %v， version %v",
					res.GroupVersionResourceKind.Group,
					res.GroupVersionResourceKind.Version, ver.String())
			}
			resources = append(resources, res)
		}
	}
	return resources, nil
}

func (m *ClustersManager) buildManager(dbCluster manager.ICluster) (*ClusterManager, error) {
	clusterName := dbCluster.GetName()
	apiServer, err := dbCluster.GetAPIServer()
	if err != nil {
		return nil, errors.Wrapf(err, "get cluster %s apiServer", clusterName)
	}
	kubeconfig, err := dbCluster.GetKubeconfig()
	if err != nil {
		return nil, errors.Wrapf(err, "get cluster %s kubeconfig", clusterName)
	}
	clientSet, config, err := BuildClient(apiServer, kubeconfig)
	if err != nil {
		return nil, errors.Wrapf(err, "build cluster %s kubernetes client", clusterName)
	}
	kubeconfigPath, err := BuildKubeConfigPath(dbCluster, kubeconfig)
	if err != nil {
		return nil, errors.Wrapf(err, "build cluster %s kubeconfig path", clusterName)
	}
	dc := clientSet.Discovery()
	resources, err := m.parseAPIResources(dc)
	if err != nil {
		return nil, errors.Wrapf(err, "parseAPIResources for cluster %s", clusterName)
	}
	restMapperRes, err := restmapper.GetAPIGroupResources(dc)
	if err != nil {
		return nil, errors.Wrapf(err, "get cluster %s api group resources", clusterName)
	}
	restMapper := restmapper.NewDiscoveryRESTMapper(restMapperRes)
	dclient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrapf(err, "build cluster %s dynamic client", clusterName)
	}
	cacheFactory, err := buildCacheController(dbCluster, clientSet, dclient, resources)
	if err != nil {
		return nil, errors.Wrapf(err, "build cluster %s cache controller", clusterName)
	}
	resHandler, err := NewResourceHandler(clientSet, dclient, restMapper, cacheFactory)
	if err != nil {
		return nil, errors.Wrapf(err, "build cluster %s resource handler error: %v", clusterName, err)
	}
	cliv2, err := clientv2.NewClient(kubeconfig)
	if err != nil {
		return nil, errors.Wrapf(err, "build cluster %s client v2", clusterName)
	}
	clusterManager := &ClusterManager{
		Cluster:        dbCluster,
		Config:         config,
		KubeClient:     resHandler,
		APIServer:      apiServer,
		KubeConfig:     kubeconfig,
		kubeConfigPath: kubeconfigPath,
		ClientV2:       cliv2,
	}
	return clusterManager, nil
}
