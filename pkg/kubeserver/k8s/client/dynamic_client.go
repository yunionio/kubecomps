package client

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

type DynamicClient struct {
	config     *rest.Config
	dynamicCli dynamic.Interface
	mapper     meta.RESTMapper
}

func NewDynamicClient(config *rest.Config) (*DynamicClient, error) {
	dcli, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	groupResources, err := restmapper.GetAPIGroupResources(clientset.Discovery())
	if err != nil {
		return nil, err
	}
	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)
	if err != nil {
		return nil, err
	}
	return &DynamicClient{
		config:     config,
		dynamicCli: dcli,
		mapper:     mapper,
	}, nil
}

func (c *DynamicClient) Update(namespace string, obj runtime.Object) (*unstructured.Unstructured, error) {
	gvk := obj.GetObjectKind().GroupVersionKind()
	gk := gvk.GroupKind()
	mapping, err := c.mapper.RESTMapping(gk, gvk.Version)
	if err != nil {
		return nil, err
	}
	unstructureObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}
	return c.dynamicCli.Resource(mapping.Resource).Namespace(namespace).Update(&unstructured.Unstructured{unstructureObj}, metav1.UpdateOptions{})
}
