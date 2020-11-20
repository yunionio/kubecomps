package types

import (
	"yunion.io/x/pkg/errors"
)

const (
	EnvKeyNodeName = "NODENAME"
)

type NodeConfig struct {
	NodeName          string      `json:"nodeName"`
	IPPools           NodeIPPools `json:"ipPools"`
	ProxyARPInterface string      `json:"proxyARPInterface"`
}

func (conf NodeConfig) Validate() error {
	if conf.NodeName == "" {
		return errors.Error("nodeName is empty")
	}

	if len(conf.IPPools) == 0 {
		return errors.Error("ipPools is empty")
	}
	for idx, pool := range conf.IPPools {
		if err := pool.Validate(); err != nil {
			return errors.Wrapf(err, "the %d ipPool", idx)
		}
	}

	return nil
}
