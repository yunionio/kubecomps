package api

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// ResourceVerber is responsible for performing generic CRUD operations on all supported resources.
type ResourceVerber interface {
	Put(kind string, namespaceSet bool, namespace string, name string,
		object runtime.Object) error
	Get(kind string, namespaceSet bool, namespace string, name string) (runtime.Object, error)
	Delete(kind string, namespaceSet bool, namespace string, name string) error
}
