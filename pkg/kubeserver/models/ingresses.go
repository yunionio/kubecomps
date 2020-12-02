package models

import (
	extensions "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
)

var (
	ingressManager *SIngressManager
)

func init() {
	GetIngressManager()
}

func GetIngressManager() *SIngressManager {
	if ingressManager == nil {
		ingressManager = NewK8sNamespaceModelManager(func() ISyncableManager {
			return &SIngressManager{
				SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
					new(SIngress),
					"ingresses_tbl",
					"ingress",
					"ingresses",
					api.ResourceNameIngress,
					extensions.GroupName,
					extensions.SchemeGroupVersion.Version,
					api.KindNameIngress,
					new(extensions.Ingress),
				),
			}
		}).(*SIngressManager)
	}
	return ingressManager
}

type SIngressManager struct {
	SNamespaceResourceBaseManager
}

type SIngress struct {
	SNamespaceResourceBase
}

func (m *SIngressManager) NewRemoteObjectForCreate(model IClusterModel, cli *client.ClusterManager, data jsonutils.JSONObject) (interface{}, error) {
	input := new(api.IngressCreateInputV2)
	data.Unmarshal(input)
	objMeta, err := input.ToObjectMeta(model.(api.INamespaceGetter))
	if err != nil {
		return nil, err
	}
	ing := &extensions.Ingress{
		ObjectMeta: objMeta,
		Spec:       input.IngressSpec,
	}
	return ing, nil
}

func (obj *SIngress) getEndpoints(ingress *extensions.Ingress) []api.Endpoint {
	endpoints := make([]api.Endpoint, 0)
	if len(ingress.Status.LoadBalancer.Ingress) > 0 {
		for _, status := range ingress.Status.LoadBalancer.Ingress {
			endpoint := api.Endpoint{Host: status.IP}
			endpoints = append(endpoints, endpoint)
		}
	}
	return endpoints
}

func (obj *SIngress) GetDetails(
	cli *client.ClusterManager,
	base interface{},
	k8sObj runtime.Object,
	isList bool,
) interface{} {
	ing := k8sObj.(*extensions.Ingress)
	detail := api.IngressDetailV2{
		NamespaceResourceDetail: obj.SNamespaceResourceBase.GetDetails(cli, base, k8sObj, isList).(api.NamespaceResourceDetail),
		Endpoints:               obj.getEndpoints(ing),
	}
	if isList {
		return detail
	}
	detail.Spec = ing.Spec
	detail.Status = ing.Status
	return detail
}
