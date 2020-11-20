package types

import (
	"fmt"

	cnet "github.com/projectcalico/libcalico-go/lib/net"

	"yunion.io/x/pkg/errors"
)

const (
	LabelManaged           = "yunion.io/managed"
	LabelManagedValueAgent = "calico-node-agent"
)

type NodeIPPool struct {
	// The node ip pool CIDR.
	CIDR string `json:"cidr"`
}

func (pool NodeIPPool) Validate() error {
	if pool.CIDR == "" {
		return errors.Errorf("CIDR is empty")
	}

	_, _, err := pool.GetIPAndNet()
	if err != nil {
		return errors.Wrap(err, "Get pool IPAndNet")
	}

	return nil
}

func (pool NodeIPPool) GetCIDR() (string, error) {
	if err := pool.Validate(); err != nil {
		return "", err
	}

	ip, ipNet, err := pool.GetIPAndNet()
	if err != nil {
		return "", err
	}

	maskLen, _ := ipNet.Mask.Size()

	return fmt.Sprintf("%s/%d", ip.To4().String(), maskLen), nil
}

func (pool NodeIPPool) GetIPAndNet() (*cnet.IP, *cnet.IPNet, error) {
	ip, ipnet, err := cnet.ParseCIDROrIP(pool.CIDR)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "ParseCIDROrIP %s", pool.CIDR)
	}
	return ip, ipnet, nil
}

type NodeIPPools []NodeIPPool
