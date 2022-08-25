package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/client/api"
)

type ResourceHandler interface {
	Create(kind string, namespace string, object *runtime.Unknown) (*runtime.Unknown, error)
	CreateV2(kind string, namespace string, object runtime.Object) (runtime.Object, error)
	Update(kind string, namespace string, name string, object *runtime.Unknown) (*runtime.Unknown, error)
	UpdateV2(kind string, object runtime.Object) (runtime.Object, error)
	Get(kind string, namespace string, name string) (runtime.Object, error)
	List(kind string, namespace string, labelSelector string) ([]runtime.Object, error)
	Delete(kind string, namespace string, name string, options *metav1.DeleteOptions) error
	GetIndexer() *CacheFactory
	GetClientset() *kubernetes.Clientset
	Close()

	Dynamic(groupKind schema.GroupKind, versions ...string) (dynamic.NamespaceableResourceInterface, error)
	DynamicGet(gvr schema.GroupVersionKind, namespace string, name string) (runtime.Object, error)
	GetResourceByKind(kind string) (api.ResourceMap, error)

	EnableBidirectionalSync()
	DisableBidirectionalSync()
}

type resourceHandler struct {
	client       *kubernetes.Clientset
	cacheFactory *CacheFactory

	// dynamic client
	restMapper    meta.RESTMapper
	dynamicClient dynamic.Interface
}

func NewResourceHandler(
	kubeClient *kubernetes.Clientset,
	dynamicClient dynamic.Interface,
	restMapper meta.RESTMapper,
	cacheFactory *CacheFactory) (ResourceHandler, error) {
	return &resourceHandler{
		client:        kubeClient,
		cacheFactory:  cacheFactory,
		restMapper:    restMapper,
		dynamicClient: dynamicClient,
	}, nil
}

func (h *resourceHandler) EnableBidirectionalSync() {
	h.cacheFactory.EnableBidirectionalSync()
}

func (h *resourceHandler) DisableBidirectionalSync() {
	h.cacheFactory.DisableBidirectionalSync()
}

func (h *resourceHandler) GetClientset() *kubernetes.Clientset {
	return h.client
}

func (h *resourceHandler) GetIndexer() *CacheFactory {
	return h.cacheFactory
}

func (h *resourceHandler) Close() {
	// TODO: figure out the root cause of panic
	utilruntime.ReallyCrash = false
	log.Errorf("============close called")
	defer utilruntime.HandleCrash()
	close(h.cacheFactory.stopChan)
	log.Errorf("====close called finish")
}

func (h *resourceHandler) Create(kind string, namespace string, object *runtime.Unknown) (*runtime.Unknown, error) {
	kubeClient, resourceMap, err := h.getClientByGroupVersion(kind)
	if err != nil {
		return nil, errors.Wrap(err, "handle resource create")
	}
	req := kubeClient.Post().
		Resource(kind).
		SetHeader("Content-Type", "application/json").
		Body([]byte(object.Raw))
	if resourceMap.Namespaced {
		req.Namespace(namespace)
	}
	var result runtime.Unknown
	err = req.Do(context.Background()).Into(&result)

	return &result, err
}

func (h *resourceHandler) CreateV2(kind string, namespace string, object runtime.Object) (runtime.Object, error) {
	kubeClient, resourceMap, err := h.getClientByGroupVersion(kind)
	if err != nil {
		return nil, errors.Wrap(err, "resourceHandler create(v2)")
	}
	req := kubeClient.Post().Resource(kind)
	if resourceMap.Namespaced {
		req.Namespace(namespace)
	}
	return req.VersionedParams(&metav1.CreateOptions{}, metav1.ParameterCodec).
		Body(object).
		Do(context.Background()).
		Get()
}

func (h *resourceHandler) Update(kind string, namespace string, name string, object *runtime.Unknown) (*runtime.Unknown, error) {
	kubeClient, resourceMap, err := h.getClientByGroupVersion(kind)
	if err != nil {
		return nil, errors.Wrap(err, "handle resource update")
	}
	req := kubeClient.Put().
		Resource(kind).
		Name(name).
		SetHeader("Content-Type", "application/json").
		Body([]byte(object.Raw))
	if resourceMap.Namespaced {
		req.Namespace(namespace)
	}

	var result runtime.Unknown
	err = req.Do(context.Background()).Into(&result)

	return &result, err
}

func (h *resourceHandler) UpdateV2(kind string, object runtime.Object) (runtime.Object, error) {
	metaObj, ok := object.(metav1.Object)
	if !ok {
		return nil, fmt.Errorf("object %#v not metav1.Object", object)
	}
	putSpec := runtime.Unknown{}
	objStr, err := json.Marshal(object)
	if err != nil {
		return nil, err
	}

	if err := json.NewDecoder(strings.NewReader(string(objStr))).Decode(&putSpec); err != nil {
		return nil, err
	}

	// todo: fix convert unknown to runtime.object
	updateObj, err := h.Update(kind, metaObj.GetNamespace(), metaObj.GetName(), &putSpec)
	if err != nil {
		return nil, err
	}
	jBytes, err := updateObj.MarshalJSON()
	if err != nil {
		return nil, err
	}
	if err := json.NewDecoder(strings.NewReader(string(jBytes))).Decode(object); err != nil {
		return nil, err
	}
	return object, err
}

func (h *resourceHandler) Delete(kind string, namespace string, name string, options *metav1.DeleteOptions) error {
	kubeClient, resourceMap, err := h.getClientByGroupVersion(kind)
	if err != nil {
		return errors.Wrap(err, " resourceHandlet delete")
	}
	req := kubeClient.Delete().
		Resource(kind).
		Name(name).
		Body(options)
	if resourceMap.Namespaced {
		req.Namespace(namespace)
	}

	return req.Do(context.Background()).Error()
}

// Get object from cache
func (h *resourceHandler) Get(kind string, namespace string, name string) (runtime.Object, error) {
	genericInformer, resource, err := h.getGenericInformer(kind)
	if err != nil {
		return nil, errors.Wrap(err, "getGenericInformer when get")
	}

	lister := genericInformer.Lister()
	var result runtime.Object
	if resource.Namespaced {
		result, err = lister.ByNamespace(namespace).Get(name)
		if err != nil {
			return nil, err
		}
	} else {
		result, err = lister.Get(name)
		if err != nil {
			return nil, err
		}
	}
	result.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
		Group:   resource.GroupVersionResourceKind.Group,
		Version: resource.GroupVersionResourceKind.Version,
		Kind:    resource.GroupVersionResourceKind.Kind,
	})

	return result, nil
}

// Dynamic return dynamic interface
func (h *resourceHandler) Dynamic(groupKind schema.GroupKind, versions ...string) (dynamic.NamespaceableResourceInterface, error) {
	mapping, err := h.restMapper.RESTMapping(groupKind, versions...)
	if err != nil {
		return nil, errors.Wrapf(err, "RESTMapping for %#v", groupKind)
	}
	return h.dynamicClient.Resource(mapping.Resource), nil
}

// DynamicGet object from cache
func (h *resourceHandler) DynamicGet(gvk schema.GroupVersionKind, namespace string, name string) (runtime.Object, error) {
	mapping, err := h.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}
	genericInformer := h.cacheFactory.dynamicInformerFactory.ForResource(mapping.Resource)
	lister := genericInformer.Lister()
	var result runtime.Object
	if namespace != "" {
		result, err = lister.ByNamespace(namespace).Get(name)
		if err != nil {
			return nil, err
		}
	} else {
		result, err = lister.Get(name)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (h *resourceHandler) getGenericInformer(kind string) (informers.GenericInformer, api.ResourceMap, error) {
	var (
		genericInformer informers.GenericInformer
		resource        api.ResourceMap
		err             error
		ok              bool
	)

	resource, ok = api.KindToResourceMap[kind]
	if !ok {
		gvkr := h.cacheFactory.GetGVKR(kind)
		if gvkr == nil {
			return nil, resource, fmt.Errorf("Resource kind (%s) not support yet.", kind)
		}
		resource = *gvkr
		genericInformer = h.cacheFactory.dynamicInformerFactory.ForResource(gvkr.GroupVersionResourceKind.GroupVersionResource)
	} else {
		gvr := resource.GroupVersionResourceKind.GroupVersionResource
		genericInformer, err = h.cacheFactory.sharedInformerFactory.ForResource(gvr)
		if err != nil {
			return nil, resource, errors.Wrapf(err, "sharedInformerFactory for resource: %#v", gvr)
		}
	}
	return genericInformer, resource, err
}

// Get object from cache
func (h *resourceHandler) List(kind string, namespace string, labelSelector string) ([]runtime.Object, error) {
	genericInformer, resource, err := h.getGenericInformer(kind)
	if err != nil {
		return nil, errors.Wrap(err, "getGenericInformer when list")
	}

	selectors, err := labels.Parse(labelSelector)
	if err != nil {
		log.Errorf("Build label selector error: %v.", err)
		return nil, err
	}

	lister := genericInformer.Lister()
	var objs []runtime.Object
	if resource.Namespaced {
		objs, err = lister.ByNamespace(namespace).List(selectors)
		if err != nil {
			return nil, err
		}
	} else {
		objs, err = lister.List(selectors)
		if err != nil {
			return nil, err
		}
	}

	for i := range objs {
		objs[i].GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
			Group:   resource.GroupVersionResourceKind.Group,
			Version: resource.GroupVersionResourceKind.Version,
			Kind:    resource.GroupVersionResourceKind.Kind,
		})
	}

	return objs, nil
}

// LListNoCache list objects from k8s directly
/*func (h *resourceHandler) ListNoCache(kind string, namespace string, labelSelector string) ([]runtime.Object, error) {
	resource, ok := api.KindToResourceMap[kind]
	if !ok {
		return nil, fmt.Errorf("Resource kind (%s) not support yet.", kind)
	}
	kubeClient := h.getClientByGroupVersion(resource.GroupVersionResourceKind.GroupVersionResource)
	req := kubeClient.Get().Resource(kind).Timeout

	if resource.Namespaced {
		req.Namespace(namespace)
	}

	for i := range objs {
		objs[i].GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{
			Group:   resource.GroupVersionResourceKind.Group,
			Version: resource.GroupVersionResourceKind.Version,
			Kind:    resource.GroupVersionResourceKind.Kind,
		})
	}

	return objs, nil
}*/
