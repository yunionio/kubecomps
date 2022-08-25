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
	"yunion.io/x/pkg/errors"
)

func (h *resourceHandler) GetResourceByKind(kind string) (api.ResourceMap, error) {
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

func (h *resourceHandler) getClientByGroupVersion(kind string) (rest.Interface, api.ResourceMap, error) {
	var (
		resource api.ResourceMap
		group    string
		version  string
		err      error
	)

	resource, err = h.GetResourceByKind(kind)
	if err != nil {
		return nil, resource, errors.Wrap(err, "GetResourceByKind")
	}

	version = resource.GroupVersionResourceKind.Version
	group = resource.GroupVersionResourceKind.Group

	switch group {
	case corev1.GroupName:
		return h.client.CoreV1().RESTClient(), resource, nil
	case apps.GroupName:
		if version == "v1beta2" {
			return h.client.AppsV1beta2().RESTClient(), resource, nil
		}
		if version == "v1" {
			return h.client.AppsV1().RESTClient(), resource, nil
		}
		return h.client.AppsV1beta1().RESTClient(), resource, nil
	case autoscalingv1.GroupName:
		return h.client.AutoscalingV1().RESTClient(), resource, nil
	case batchv1.GroupName:
		if version == "v1beta1" {
			return h.client.BatchV1beta1().RESTClient(), resource, nil
		}
		return h.client.BatchV1().RESTClient(), resource, nil
	case extensionsv1beta1.GroupName:
		return h.client.ExtensionsV1beta1().RESTClient(), resource, nil
	case storagev1.GroupName:
		return h.client.StorageV1().RESTClient(), resource, nil
	case rbacv1.GroupName:
		return h.client.RbacV1().RESTClient(), resource, nil
	case networking.GroupName:
		if version == "v1beta1" {
			return h.client.NetworkingV1beta1().RESTClient(), resource, nil
		}
		return h.client.NetworkingV1().RESTClient(), resource, nil
	default:
		log.Warningf("could not match any exist group, return a default client")
		return h.client.CoreV1().RESTClient(), resource, nil
	}
}
