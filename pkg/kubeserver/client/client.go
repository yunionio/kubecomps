package client

import (
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	// "k8s.io/apimachinery/pkg/api/meta"
	// "k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	// "k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/clientv2"
	"yunion.io/x/kubecomps/pkg/kubeserver/models/manager"
)

const (
	// High enough QPS to fit all expected use cases.
	defaultQPS = 1e6
	// High enough Burst to fit all expected use cases.
	defaultBurst = 1e6
	// full resync cache resource time
	defaultResyncPeriod = 30 * time.Second
)

var (
	ErrNotExist = errors.Error("cluster not exist.")
	ErrStatus   = errors.Error("cluster invalid status, please try again later.")
)

type ClusterManager struct {
	Cluster    manager.ICluster
	Config     *rest.Config
	KubeClient ResourceHandler
	APIServer  string
	KubeConfig string
	// KubeConfigPath used for kubectl or helm client
	kubeConfigPath string
	ClientV2       *clientv2.Client
}

func (c ClusterManager) GetKubeConfigPath() (string, error) {
	if c.kubeConfigPath == "" {
		confPath, err := BuildKubeConfigPath(c.Cluster, c.KubeConfig)
		if err != nil {
			return "", err
		}
		c.kubeConfigPath = confPath
	}
	if _, err := os.Stat(c.kubeConfigPath); err != nil {
		if os.IsNotExist(err) {
			confPath, err := BuildKubeConfigPath(c.Cluster, c.KubeConfig)
			if err != nil {
				return "", err
			}
			c.kubeConfigPath = confPath
		} else {
			return "", err
		}
	}
	return c.kubeConfigPath, nil
}

func (c ClusterManager) GetClusterObject() manager.ICluster {
	return c.Cluster
}

func (c ClusterManager) GetId() string {
	return c.Cluster.GetId()
}

func (c ClusterManager) GetName() string {
	return c.Cluster.GetName()
}

func (c ClusterManager) GetIndexer() *CacheFactory {
	return c.KubeClient.GetIndexer()
}

func (c ClusterManager) GetClientset() kubernetes.Interface {
	return c.KubeClient.GetClientset()
}

func (c ClusterManager) GetClient() *clientv2.Client {
	return c.ClientV2
}

func (c ClusterManager) GetHandler() ResourceHandler {
	return c.KubeClient
}

func (c ClusterManager) Close() {
	os.RemoveAll(c.kubeConfigPath)
	c.KubeClient.Close()
}

func GetManagerByCluster(c manager.ICluster) (*ClusterManager, error) {
	return GetManager(c.GetId())
}

func GetManager(cluster string) (*ClusterManager, error) {
	// manInterface, exist := clusterManagerSets.Load(cluster)
	man := clustersManager.getManager(cluster)
	if man == nil {
		// BuildApiserverClient()
		// manInterface, exist = clusterManagerSets.Load(cluster)
		// if !exist {
		// return nil, errors.Wrapf(ErrNotExist, "cluster %s", cluster)
		// }
		return nil, errors.Wrapf(ErrNotExist, "cluster %s", cluster)
	}
	status := man.Cluster.GetStatus()
	if status != api.ClusterStatusRunning {
		return nil, errors.Wrapf(ErrStatus, "cluster %s status %s", cluster, status)
	}
	return man, nil
}

func BuildClientConfig(master string, kubeconfig string) (*rest.Config, *clientcmdapi.Config, error) {
	configInternal, err := clientcmd.Load([]byte(kubeconfig))
	if err != nil {
		return nil, nil, err
	}
	curCtxName := configInternal.CurrentContext
	curCtx, ok := configInternal.Contexts[curCtxName]
	if !ok {
		return nil, nil, errors.Errorf("Not found context %q", curCtxName)
	}
	ctxClsName := curCtx.Cluster
	cls, ok := configInternal.Clusters[ctxClsName]
	if !ok {
		return nil, nil, errors.Errorf("Not found cluster %q", ctxClsName)
	}
	cls.Server = master
	configInternal.Clusters[ctxClsName] = cls

	clientConfig := clientcmd.NewDefaultClientConfig(*configInternal, &clientcmd.ConfigOverrides{
		ClusterDefaults: clientcmdapi.Cluster{Server: master},
	})
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, nil, errors.Wrap(err, "build client rest config")
	}
	restConfig.QPS = defaultQPS
	restConfig.Burst = defaultBurst
	apiConfig, err := clientConfig.RawConfig()
	if err != nil {
		return nil, nil, errors.Wrap(err, "build client api raw config")
	}
	return restConfig, &apiConfig, nil
}

func BuildClient(master string, kubeconfig string) (*kubernetes.Clientset, *rest.Config, error) {
	restConfig, _, err := BuildClientConfig(master, kubeconfig)
	if err != nil {
		log.Errorf("build client config error. %v ", err)
		return nil, nil, err
	}

	clientSet, err := kubernetes.NewForConfig(restConfig)

	if err != nil {
		log.Errorf("(%s) kubernetes.NewForConfig(%v) error.%v", master, err, restConfig)
		return nil, nil, err
	}

	return clientSet, restConfig, nil
}

func ClusterKubeConfigPath(c manager.ICluster) string {
	return path.Join("/tmp", strings.Join([]string{"kubecluster", c.GetName(), c.GetId(), ".kubeconfig"}, "-"))
}

func BuildKubeConfigPath(c manager.ICluster, kubeconfig string) (string, error) {
	configPath := ClusterKubeConfigPath(c)
	if err := ioutil.WriteFile(configPath, []byte(kubeconfig), 0666); err != nil {
		return "", errors.Wrapf(err, "write %s", configPath)
	}
	return configPath, nil
}
