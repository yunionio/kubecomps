package common

import (
	"bytes"

	"k8s.io/api/core/v1"

	api "yunion.io/x/kubecomps/pkg/kubeserver/api"
)

// GetExternalEndpoints returns endpoints that are externally reachable for a service.
func GetExternalEndpoints(service *v1.Service) []api.Endpoint {
	var externalEndpoints []api.Endpoint
	if service.Spec.Type == v1.ServiceTypeLoadBalancer {
		for _, ingress := range service.Status.LoadBalancer.Ingress {
			externalEndpoints = append(externalEndpoints, getExternalEndpoint(ingress, service.Spec.Ports))
		}
	}

	for _, ip := range service.Spec.ExternalIPs {
		externalEndpoints = append(externalEndpoints, api.Endpoint{
			Host:  ip,
			Ports: GetServicePorts(service.Spec.Ports),
		})
	}

	return externalEndpoints
}

// GetInternalEndpoint returns internal endpoint name for the given service properties, e.g.,
// "my-service.namespace 80/TCP" or "my-service 53/TCP,53/UDP".
func GetInternalEndpoint(serviceName, namespace string, ports []v1.ServicePort) api.Endpoint {
	name := serviceName

	if namespace != v1.NamespaceDefault && len(namespace) > 0 && len(serviceName) > 0 {
		bufferName := bytes.NewBufferString(name)
		bufferName.WriteString(".")
		bufferName.WriteString(namespace)
		name = bufferName.String()
	}

	return api.Endpoint{
		Host:  name,
		Ports: GetServicePorts(ports),
	}
}

// Returns external endpoint name for the given service properties.
func getExternalEndpoint(ingress v1.LoadBalancerIngress, ports []v1.ServicePort) api.Endpoint {
	var host string
	if ingress.Hostname != "" {
		host = ingress.Hostname
	} else {
		host = ingress.IP
	}
	return api.Endpoint{
		Host:  host,
		Ports: GetServicePorts(ports),
	}
}
