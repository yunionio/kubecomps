package models

import (
	"errors"
	extensions "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/kubernetes/pkg/apis/networking"
	"strconv"
	"yunion.io/x/kubecomps/pkg/kubeserver/k8s/common/model"
	"yunion.io/x/log"

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
					new(unstructured.Unstructured),
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

func (m *SIngressManager) GetK8sResourceInfo(serverVersion *version.Info) model.K8sResourceInfo {
	if serverVersion != nil {
		if minor, err := strconv.Atoi(serverVersion.Minor); err == nil && minor >= 21 {
			return model.K8sResourceInfo{
				ResourceName: api.ResourceNameIngress,
				Group:        networking.GroupName,
				Version:      "v1",
				KindName:     api.KindNameIngress,
				Object:       new(networking.Ingress),
			}
		}
	}
	return model.K8sResourceInfo{
		ResourceName: api.ResourceNameIngress,
		Group:        extensions.GroupName,
		Version:      extensions.SchemeGroupVersion.Version,
		KindName:     api.KindNameIngress,
		Object:       new(extensions.Ingress),
	}
}

func (m *SIngressManager) NewRemoteObjectForCreate(model IClusterModel, cli *client.ClusterManager, data jsonutils.JSONObject) (interface{}, error) {
	version := getIngressVersion(cli)

	if version == 3 {
		input := new(api.IngressCreateInputV3)
		data.Unmarshal(input)
		objMeta, err := input.ToObjectMeta(model.(api.INamespaceGetter))
		if err != nil {
			return nil, err
		}
		ing := &networking.Ingress{
			ObjectMeta: objMeta,
			Spec:       input.IngressSpec,
		}
		return ing, nil
	} else if version == 2 {
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
	} else {
		log.Errorln("Unsupported version of ingress")
	}

	return nil, errors.New("unsupported version of ingress")
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

func (obj *SIngress) getEndpointsV3(ingress *networking.Ingress) []api.Endpoint {
	endpoints := make([]api.Endpoint, 0)
	if len(ingress.Status.LoadBalancer.Ingress) > 0 {
		for _, status := range ingress.Status.LoadBalancer.Ingress {
			endpoint := api.Endpoint{Host: status.IP}
			endpoints = append(endpoints, endpoint)
		}
	}
	return endpoints
}

func getIngressVersion(cli *client.ClusterManager) int {
	serverVersion, err := cli.GetClientset().Discovery().ServerVersion()
	if err != nil {
		return 0
	}
	if minor, err := strconv.Atoi(serverVersion.Minor); err == nil && minor >= 21 {
		return 3
	}
	return 2
}

func (obj *SIngress) GetDetails(
	cli *client.ClusterManager,
	base interface{},
	k8sObj runtime.Object,
	isList bool,
) interface{} {
	version := getIngressVersion(cli)

	if version == 3 {
		ing := k8sObj.(*networking.Ingress)
		detail := api.IngressDetailV3{
			NamespaceResourceDetail: obj.SNamespaceResourceBase.GetDetails(cli, base, k8sObj, isList).(api.NamespaceResourceDetail),
			Endpoints:               obj.getEndpointsV3(ing),
		}
		if isList {
			return detail
		}
		detail.Spec = ing.Spec
		detail.Status = ing.Status
		return detail
	} else if version == 2 {
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
	} else {
		log.Errorln("Unsupported version of ingress")
	}

	return nil
}
