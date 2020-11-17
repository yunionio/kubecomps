package metadata

import (
	"net"

	"yunion.io/x/jsonutils"
)

// Digest contains the parts of a meta-data info of instance
type Digest struct {
	Hostname          string
	LocalIPv4         net.IP
	PublicIPv4        net.IP
	NetworkInterfaces []NetworkInterface
}

type NetworkInterface struct {
	Mac        net.HardwareAddr
	PrimaryIP  net.IP
	PrivateIPs []net.IP
	PublicIPs  []net.IP
}

func (d Digest) String() string {
	return jsonutils.Marshal(d).String()
}

func (d Digest) PrettyString() string {
	return jsonutils.Marshal(d).PrettyString()
}
