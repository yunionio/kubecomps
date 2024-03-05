package api

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
)

// ResourceVerber is responsible for performing generic CRUD operations on all supported resources.
type ResourceVerber interface {
	Put(ctx context.Context, kind string, namespaceSet bool, namespace string, name string,
		object runtime.Object) error
	Get(ctx context.Context, kind string, namespaceSet bool, namespace string, name string) (runtime.Object, error)
	Delete(ctx context.Context, kind string, namespaceSet bool, namespace string, name string) error
}
