package client

import (
	//"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	restclient "k8s.io/client-go/rest"

	clientapi "yunion.io/x/kubecomps/pkg/kubeserver/k8s/client/api"
	api "yunion.io/x/kubecomps/pkg/kubeserver/types/apis"
)

// resourceVerber is a struct responsible for doing common verb operations on resources, like
// DELETE, PUT, UPDATE
type resourceVerber struct {
	client            RESTClient
	extensionsClient  RESTClient
	appsClient        RESTClient
	batchClient       RESTClient
	betaBatchClient   RESTClient
	autoscalingClient RESTClient
	storageClient     RESTClient
}

func (verber *resourceVerber) getRESTClientByType(clientType api.ClientType) RESTClient {
	switch clientType {
	case api.ClientTypeExtensionClient:
		return verber.extensionsClient
	case api.ClientTypeAppsClient:
		return verber.appsClient
	case api.ClientTypeBatchClient:
		return verber.batchClient
	case api.ClientTypeBetaBatchClient:
		return verber.betaBatchClient
	case api.ClientTypeAutoscalingClient:
		return verber.autoscalingClient
	case api.ClientTypeStorageClient:
		return verber.storageClient
	default:
		return verber.client
	}
}

// RESTClient is an interface for REST operations used in this file.
type RESTClient interface {
	Delete() *restclient.Request
	Put() *restclient.Request
	Get() *restclient.Request
}

// NewResourceVerber creates a new resource verber that uses the given client for performing operations.
func NewResourceVerber(client, extensionsClient, appsClient,
	batchClient, betaBatchClient, autoscalingClient, storageClient RESTClient) clientapi.ResourceVerber {
	return &resourceVerber{client, extensionsClient, appsClient,
		batchClient, betaBatchClient, autoscalingClient, storageClient}
}

// Delete deletes the resource of the given kind in the given namespace with the given name.
func (verber *resourceVerber) Delete(kind string, namespaceSet bool, namespace string, name string) error {
	resourceSpec, ok := api.KindToAPIMapping[kind]
	if !ok {
		return fmt.Errorf("Unknown resource kind: %s", kind)
	}

	if namespaceSet != resourceSpec.Namespaced {
		if namespaceSet {
			return fmt.Errorf("Set namespace for not-namespaced resource kind: %s", kind)
		} else {
			return fmt.Errorf("Set no namespace for namespaced resource kind: %s", kind)
		}
	}

	client := verber.getRESTClientByType(resourceSpec.ClientType)

	// Do cascade delete by default, as this is what users typically expect.
	defaultPropagationPolicy := v1.DeletePropagationForeground
	defaultDeleteOptions := &v1.DeleteOptions{
		PropagationPolicy: &defaultPropagationPolicy,
	}

	req := client.Delete().Resource(resourceSpec.Resource).Name(name).Body(defaultDeleteOptions)

	if resourceSpec.Namespaced {
		req.Namespace(namespace)
	}

	return req.Do().Error()
}

// Put puts new resource version of the given kind in the given namespace with the given name.
func (verber *resourceVerber) Put(kind string, namespaceSet bool, namespace string, name string,
	obj runtime.Object) error {

	resourceSpec, ok := api.KindToAPIMapping[kind]
	if !ok {
		return fmt.Errorf("Unknown resource kind: %s", kind)
	}

	if namespaceSet != resourceSpec.Namespaced {
		if namespaceSet {
			return fmt.Errorf("Set namespace for not-namespaced resource kind: %s", kind)
		} else {
			return fmt.Errorf("Set no namespace for namespaced resource kind: %s", kind)
		}
	}

	client := verber.getRESTClientByType(resourceSpec.ClientType)

	//objBytes, err := json.Marshal(obj)
	//if err != nil {
	//return fmt.Errorf("Marshal obj %#v: %v", obj, err)
	//}
	req := client.Put().
		Resource(resourceSpec.Resource).
		Name(name).
		//SetHeader("Content-Type", "application/json").
		Body(obj)

	if resourceSpec.Namespaced {
		req.Namespace(namespace)
	}

	return req.Do().Error()
}

// Get gets the resource of the given kind in the given namespace with the given name.
func (verber *resourceVerber) Get(kind string, namespaceSet bool, namespace string, name string) (runtime.Object, error) {
	resourceSpec, ok := api.KindToAPIMapping[kind]
	if !ok {
		return nil, fmt.Errorf("Unknown resource kind: %s", kind)
	}

	if namespaceSet != resourceSpec.Namespaced {
		if namespaceSet {
			return nil, fmt.Errorf("Set namespace for not-namespaced resource kind: %s", kind)
		} else {
			return nil, fmt.Errorf("Set no namespace for namespaced resource kind: %s", kind)
		}
	}

	client := verber.getRESTClientByType(resourceSpec.ClientType)
	result := &runtime.Unknown{}
	req := client.Get().Resource(resourceSpec.Resource).Name(name).SetHeader("Accept", "application/json")

	if resourceSpec.Namespaced {
		req.Namespace(namespace)
	}

	err := req.Do().Into(result)
	return result, err
}
