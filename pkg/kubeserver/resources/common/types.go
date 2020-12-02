package common

const (
	// k8s annotations for create pod
	YUNION_CNI_NETWORK_ANNOTATION = "cni.yunion.io/network"
	YUNION_CNI_IPADDR_ANNOTATION  = "cni.yunion.io/ip"

	YUNION_LB_NETWORK_ANNOTATION = "loadbalancer.yunion.io/network"
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
