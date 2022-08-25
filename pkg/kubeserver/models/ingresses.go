package models

import (
	extensions "k8s.io/api/extensions/v1beta1"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/apis/networking"
	"yunion.io/x/jsonutils"
	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
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
					"",
					"",
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

func (m *SIngressManager) NewRemoteObjectForCreate(model IClusterModel, cli *client.ClusterManager, data jsonutils.JSONObject) (interface{}, error) {
	kind, _ := cli.GetHandler().GetResourceByKind(api.KindNameIngress)
	if kind.GroupVersionResourceKind.Group == extensions.GroupName {
		input := new(api.IngressCreateInputV2)
		err := data.Unmarshal(input)
		if err != nil {
			return nil, errors.Wrap(err, "ingress input unmarshal error")
		}
		objMeta, err := input.ToObjectMeta(model.(api.INamespaceGetter))
		if err != nil {
			return nil, errors.Wrap(err, "ingress input get meta error")
		}
		return &extensions.Ingress{
			ObjectMeta: objMeta,
			Spec:       input.IngressSpec,
		}, nil
	} else if kind.GroupVersionResourceKind.Group == networking.GroupName {
		if kind.GroupVersionResourceKind.Version == "v1beta1" {
			input := new(api.IngressCreateInputNetworkingV1beta1)
			err := data.Unmarshal(input)
			if err != nil {
				return nil, errors.Wrap(err, "ingress input unmarshal error")
			}
			objMeta, err := input.ToObjectMeta(model.(api.INamespaceGetter))
			if err != nil {
				return nil, errors.Wrap(err, "ingress input get meta error")
			}
			return &v1beta1.Ingress{
				ObjectMeta: objMeta,
				Spec:       input.IngressSpec,
			}, nil
		} else if kind.GroupVersionResourceKind.Version == "v1" {
			input := new(api.IngressCreateInputNetworkingV1)
			err := data.Unmarshal(input)
			if err != nil {
				return nil, errors.Wrap(err, "ingress input unmarshal error")
			}
			objMeta, err := input.ToObjectMeta(model.(api.INamespaceGetter))
			if err != nil {
				return nil, errors.Wrap(err, "ingress input get meta error")
			}
			return &v1.Ingress{
				ObjectMeta: objMeta,
				Spec:       input.IngressSpec,
			}, nil
		}
	}

	log.Errorf("unexpected ingress GVR info")
	return nil, nil
}

func (obj *SIngress) getEndpoints(rObj *unstructured.Unstructured) []api.Endpoint {
	endpoints := make([]api.Endpoint, 0)
	ingress, _, _ := unstructured.NestedSlice(rObj.Object, "status", "loadBalancer", "ingress")
	if len(ingress) > 0 {
		for _, status := range ingress {
			ip, _, _ := unstructured.NestedString(status.(map[string]interface{}), "ip")
			endpoint := api.Endpoint{Host: ip}
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
	ing := k8sObj.(*unstructured.Unstructured)
	detail := api.IngressDetailV2{
		NamespaceResourceDetail: obj.SNamespaceResourceBase.GetDetails(cli, base, k8sObj, isList).(api.NamespaceResourceDetail),
		Endpoints:               obj.getEndpoints(ing),
	}
	if isList {
		return detail
	}
	detail.Spec, _, _ = unstructured.NestedMap(ing.Object, "spec")
	detail.Status, _, _ = unstructured.NestedMap(ing.Object, "status")
	return detail
}
