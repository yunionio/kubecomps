package common

import (
	"k8s.io/api/core/v1"

	api "yunion.io/x/kubecomps/pkg/kubeserver/api"
)

// GetServicePorts returns human readable name for the given service ports list.
func GetServicePorts(apiPorts []v1.ServicePort) []api.ServicePort {
	var ports []api.ServicePort
	for _, port := range apiPorts {
		ports = append(ports, api.ServicePort{port.Port, port.Protocol, port.NodePort})
	}
	return ports
}
