package client

import (
	"yunion.io/x/kubecomps/pkg/kubeserver/types/apis"
)

type Node struct {
	Id                string `json:"id"`
	RequestedHostname string `json:"requestedHostname,omitempty"`
	Address           string `json:"address"`
	InternalAddress   string `json:"internalAddress,omitempty"`

	DockerInfo *apis.DockerInfo `json:"dockerInfo,omitempty"`
}
