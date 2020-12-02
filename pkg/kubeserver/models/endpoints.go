package models

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
)

var (
	endpointManager *SEndpointManager
)

type SEndpointManager struct {
	SNamespaceResourceBaseManager
}

type SEndpoint struct {
	SNamespaceResourceBase
}

func GetEndpointManager() *SEndpointManager {
	if endpointManager == nil {
		endpointManager = NewK8sNamespaceModelManager(func() ISyncableManager {
			return &SEndpointManager{
				SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
					SEndpoint{},
					"endpoints_tbl",
					"k8s_endpoint",
					"k8s_endpoints",
					api.ResourceNameEndpoint,
					v1.GroupName,
					v1.SchemeGroupVersion.Version,
					api.KindNameEndpoint,
					new(v1.Endpoints),
				),
			}
		}).(*SEndpointManager)
	}
	return endpointManager
}

func (m *SEndpointManager) GetRawEndpoints(cluster *client.ClusterManager, ns string) ([]*v1.Endpoints, error) {
	indexer := cluster.GetHandler().GetIndexer()
	return indexer.EndpointLister().Endpoints(ns).List(labels.Everything())
}

func (m *SEndpointManager) GetRawEndpointsByService(cluster *client.ClusterManager, svc *v1.Service) ([]*v1.Endpoints, error) {
	eps, err := m.GetRawEndpoints(cluster, svc.GetNamespace())
	if err != nil {
		return nil, err
	}
	ret := make([]*v1.Endpoints, 0)
	for _, ip := range eps {
		if ip.Name == svc.GetName() {
			ret = append(ret, ip)
		}
	}
	return ret, nil
}
