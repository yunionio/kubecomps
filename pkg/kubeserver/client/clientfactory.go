package client

import (
	"fmt"

	apps "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
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

func (h *resourceHandler) getClientByGroupVersion(resource api.ResourceMap) (rest.Interface, error) {
	switch resource.GroupVersionResourceKind.Group {
	case corev1.GroupName:
		return h.client.CoreV1().RESTClient(), nil
	case apps.GroupName:
		if resource.GroupVersionResourceKind.Version == "v1beta2" {
			return h.client.AppsV1beta2().RESTClient(), nil
		}
		if resource.GroupVersionResourceKind.Version == "v1" {
			return h.client.AppsV1().RESTClient(), nil
		}
		return h.client.AppsV1beta1().RESTClient(), nil
	case autoscalingv1.GroupName:
		return h.client.AutoscalingV1().RESTClient(), nil
	case batchv1.GroupName:
		if resource.GroupVersionResourceKind.Version == "v1beta1" {
			return h.client.BatchV1beta1().RESTClient(), nil
		}
		return h.client.BatchV1().RESTClient(), nil
	case extensionsv1beta1.GroupName:
		return h.client.ExtensionsV1beta1().RESTClient(), nil
	case storagev1.GroupName:
		return h.client.StorageV1().RESTClient(), nil
	case rbacv1.GroupName:
		return h.client.RbacV1().RESTClient(), nil
	case networking.GroupName:
		if resource.GroupVersionResourceKind.Version == "v1beta1" {
			return h.client.NetworkingV1beta1().RESTClient(), nil
		}
		return h.client.NetworkingV1().RESTClient(), nil
	default:
		log.Warningf("could not match any exist group, return a default client")
		return h.client.CoreV1().RESTClient(), nil
	}
}
