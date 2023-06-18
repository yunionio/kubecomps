package api

import (
	"k8s.io/api/core/v1"
)

// Endpoint describes an endpoint that is host and a list of available ports for that host
type Endpoint struct {
	// Hostname, either as a domain name or IP address
	Host string `json:"host"`

	// List of ports opened for this endpoint on the hostname
	Ports []ServicePort `json:"ports"`
}

// ServicePort is a pair of port and protocol, e.g. a service endpoint.
type ServicePort struct {
	// Positive port number.
	Port int32 `json:"port"`

	// Protocol name, e.g., TCP or UDP.
	Protocol v1.Protocol `json:"protocol"`

	// The port on each node on which service is exposed.
	NodePort int32 `json:"nodePort"`

	// Nodes ip with nodePort
	NodePortEndpoints []string `json:"nodePortEndpoints"`
}

type EndpointDetail struct {
	NamespaceResourceDetail

	// Hostname, either as a domain name or IP address.
	Host string `json:"host"`

	// Name of the node the endpoint is located
	NodeName *string `json:"nodeName"`

	// Status of the endpoint
	Ready bool `json:"ready"`

	// Array of endpoint ports
	Ports []v1.EndpointPort `json:"ports"`
}
