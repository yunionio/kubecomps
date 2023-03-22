package client

import (
	"fmt"

	apps "k8s.io/api/apps/v1"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/apis/networking"
	"yunion.io/x/kubecomps/pkg/kubeserver/client/api"
	"yunion.io/x/log"
)

func (h *resourceHandler) getResourceByKind(kind string) (api.ResourceMap, error) {
	resource, ok := api.KindToResourceMap[kind]
	if !ok {
		gvkr := h.cacheFactory.GetGVKR(kind)
		if gvkr == nil {
			return resource, fmt.Errorf("resource kind (%s) not support yet", kind)
		}
		resource = *gvkr
	}

	return resource, nil
}

func (h *resourceHandler) getClientByGroupVersion(resource api.ResourceMap) rest.Interface {
	switch resource.GroupVersionResourceKind.Group {
	case corev1.GroupName:
		return h.client.CoreV1().RESTClient()
	case apps.GroupName:
		if resource.GroupVersionResourceKind.Version == "v1beta2" {
			return h.client.AppsV1beta2().RESTClient()
		}
		if resource.GroupVersionResourceKind.Version == "v1" {
			return h.client.AppsV1().RESTClient()
		}
		return h.client.AppsV1beta1().RESTClient()
	case autoscalingv2beta2.GroupName:
		if resource.GroupVersionResourceKind.Version == "v2beta1" {
			return h.client.AutoscalingV2beta1().RESTClient()
		}
		if resource.GroupVersionResourceKind.Version == "v2beta2" {
			return h.client.AutoscalingV2beta2().RESTClient()
		}
		return h.client.AutoscalingV1().RESTClient()
	case batchv1.GroupName:
		if resource.GroupVersionResourceKind.Version == "v1beta1" {
			return h.client.BatchV1beta1().RESTClient()
		}
		return h.client.BatchV1().RESTClient()
	case extensionsv1beta1.GroupName:
		return h.client.ExtensionsV1beta1().RESTClient()
	case storagev1.GroupName:
		return h.client.StorageV1().RESTClient()
	case rbacv1.GroupName:
		return h.client.RbacV1().RESTClient()
	case networking.GroupName:
		if resource.GroupVersionResourceKind.Version == "v1beta1" {
			return h.client.NetworkingV1beta1().RESTClient()
		}
		return h.client.NetworkingV1().RESTClient()
	default:
		log.Warningf("could not match any exist group, return a default client")
		return h.client.CoreV1().RESTClient()
	}
}
