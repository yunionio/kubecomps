package api

import (
	"fmt"
	"strings"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/rand"
)

type Service struct {
	ObjectMeta
	TypeMeta

	// InternalEndpoint of all kubernetes services that have the same label selector as connected Replication
	// Controller. Endpoint is DNS name merged with ports
	InternalEndpoint Endpoint `json:"internalEndpoint"`

	// ExternalEndpoints of all kubernetes services that have the same label selector as connected Replication
	// Controller. Endpoint is DNS name merged with ports
	ExternalEndpoints []Endpoint `json:"externalEndpoints"`

	// Label selector of the service
	Selector map[string]string `json:"selector"`

	// Type determines how the service will be exposed. Valid options: ClusterIP, NodePort, LoadBalancer
	Type v1.ServiceType `json:"type"`

	// ClusterIP is usually assigned by the master. Valid values are None, empty string (""), or
	// a valid IP address. None can be specified for headless services when proxying is not required
	ClusterIP string `json:"clusterIP"`
}

type ServiceDetail struct {
	Service

	// List of Endpoint obj. that are endpoints of this Service.
	Endpoints []*EndpointDetail `json:"endpoints"`

	// List of events related to this Service
	Events []*Event `json:"events"`

	// Pods represents list of pods targeted by same label selector as this service.
	Pods []*Pod `json:"pods"`

	// Show the value of the SessionAffinity of the Service.
	SessionAffinity v1.ServiceAffinity `json:"sessionAffinity"`
}

// PortMapping is a specification of port mapping for an application deployment.
type PortMapping struct {
	// Port that will be exposed on the service.
	Port int32 `json:"port"`

	// Docker image path for the application.
	TargetPort int32 `json:"targetPort"`

	// IP protocol for the mapping, e.g., "TCP" or "UDP".
	Protocol v1.Protocol `json:"protocol"`
}

func GenerateName(base string) string {
	maxNameLength := 63
	randomLength := 5
	maxGeneratedNameLength := maxNameLength - randomLength
	if len(base) > maxGeneratedNameLength {
		base = base[:maxGeneratedNameLength]
	}
	return fmt.Sprintf("%s%s", base, rand.String(randomLength))
}

func GeneratePortMappingName(portMapping PortMapping) string {
	return GenerateName(fmt.Sprintf("%s-%d-%d-", strings.ToLower(string(portMapping.Protocol)),
		portMapping.Port, portMapping.TargetPort))
}

func (p PortMapping) ToServicePort() v1.ServicePort {
	return v1.ServicePort{
		Protocol: p.Protocol,
		Port:     p.Port,
		Name:     GeneratePortMappingName(p),
		TargetPort: intstr.IntOrString{
			Type:   intstr.Int,
			IntVal: p.TargetPort,
		},
	}
}

type ServiceCreateOption struct {
	Type                string            `json:"type"`
	IsExternal          bool              `json:"isExternal"`
	PortMappings        []PortMapping     `json:"portMappings"`
	Selector            map[string]string `json:"selector"`
	LoadBalancerCluster string            `json:"loadBalancerCluster"`
	LoadBalancerNetwork string            `json:"loadBalancerNetwork"`
}

type ServiceCreateInput struct {
	NamespaceResourceCreateInput
	ServiceCreateOption
}

const (
	// k8s annotations for create pod
	YUNION_CNI_NETWORK_ANNOTATION = "cni.yunion.io/network"
	YUNION_CNI_IPADDR_ANNOTATION  = "cni.yunion.io/ip"

	YUNION_LB_NETWORK_ANNOTATION = "loadbalancer.yunion.io/network"
	YUNION_LB_CLUSTER_ANNOTATION = "loadbalancer.yunion.io/cluster"
)

type NetworkConfig struct {
	Network string `json:"network"`
	Address string `json:"address"`
}

func (n NetworkConfig) ToPodAnnotation() map[string]string {
	ret := make(map[string]string)
	if n.Network != "" {
		ret[YUNION_CNI_NETWORK_ANNOTATION] = n.Network
	}
	if n.Address != "" {
		ret[YUNION_CNI_IPADDR_ANNOTATION] = n.Address
	}
	return ret
}

// GetServicePorts returns human readable name for the given service ports list.
func GetServicePorts(apiPorts []v1.ServicePort) []ServicePort {
	var ports []ServicePort
	for _, port := range apiPorts {
		ports = append(ports, ServicePort{port.Port, port.Protocol, port.NodePort})
	}
	return ports
}

type ServiceListInput struct {
	ListInputK8SNamespaceBase
	ListInputOwner
}

type ServiceDetailV2 struct {
	NamespaceResourceDetail

	// InternalEndpoint of all kubernetes services that have the same label selector as connected Replication
	// Controller. Endpoint is DNS name merged with ports
	InternalEndpoint Endpoint `json:"internalEndpoint"`

	// ExternalEndpoints of all kubernetes services that have the same label selector as connected Replication
	// Controller. Endpoint is DNS name merged with ports
	ExternalEndpoints []Endpoint `json:"externalEndpoints"`

	// Label selector of the service
	Selector map[string]string `json:"selector"`

	// Type determines how the service will be exposed. Valid options: ClusterIP, NodePort, LoadBalancer
	Type v1.ServiceType `json:"type"`

	// ClusterIP is usually assigned by the master. Valid values are None, empty string (""), or
	// a valid IP address. None can be specified for headless services when proxying is not required
	ClusterIP string `json:"clusterIP"`

	// List of Endpoint obj. that are endpoints of this Service.
	// Endpoints []*EndpointDetail `json:"endpoints"`

	// List of events related to this Service
	// Events []*Event `json:"events"`

	// Pods represents list of pods targeted by same label selector as this service.
	Pods []*PodDetailV2 `json:"pods"`

	// Show the value of the SessionAffinity of the Service.
	SessionAffinity v1.ServiceAffinity `json:"sessionAffinity"`
}
