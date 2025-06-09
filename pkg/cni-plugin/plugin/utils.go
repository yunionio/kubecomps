package plugin

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/vishvananda/netlink"

	"yunion.io/x/pkg/errors"
)

const (
	K8S_POD_NAMESPACE          = "K8S_POD_NAMESPACE"
	K8S_POD_NAME               = "K8S_POD_NAME"
	K8S_POD_INFRA_CONTAINER_ID = "K8S_POD_INFRA_CONTAINER_ID"
	K8S_POD_UID                = "K8S_POD_UID"
)

type PodInfo struct {
	Id          string
	Name        string
	Namespace   string
	ContainerId string
}

func NewPodInfoFromCNIArgs(args string) (*PodInfo, error) {
	segs := strings.Split(args, ";")
	ret := new(PodInfo)
	for _, seg := range segs {
		kv := strings.Split(seg, "=")
		if len(kv) != 2 {
			return nil, fmt.Errorf("Invalid args part: %q", seg)
		}
		key := kv[0]
		val := kv[1]
		switch key {
		case K8S_POD_NAMESPACE:
			ret.Namespace = val
		case K8S_POD_NAME:
			ret.Name = val
		case K8S_POD_INFRA_CONTAINER_ID:
			ret.ContainerId = val
		case K8S_POD_UID:
			ret.Id = val
		}
	}
	if ret.Id == "" {
		return nil, errors.Errorf("Not found %s from args %s", K8S_POD_UID, args)
	}
	return ret, nil
}

func (p PodInfo) GetDescPath() string {
	return filepath.Join(GetCloudServerDir(), p.Id, "desc")
}

type CloudPod struct {
	*PodInfo
	desc *PodDesc
}

func GetCloudServerDir() string {
	// TODO: make it configurable
	return "/opt/cloud/workspace/servers"
}

func NewCloudPodFromCNIArgs(args string) (*CloudPod, error) {
	info, err := NewPodInfoFromCNIArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "NewPodInfoFromCNIArgs")
	}
	descFile := info.GetDescPath()
	descData, err := ioutil.ReadFile(descFile)
	if err != nil {
		return nil, errors.Wrap(err, "read desc file")
	}
	desc := new(PodDesc)
	if err := json.Unmarshal(descData, desc); err != nil {
		return nil, errors.Wrap(err, "json.Unmarshal")
	}
	pod := &CloudPod{
		PodInfo: info,
		desc:    desc,
	}
	return pod, nil
}

func (p CloudPod) GetDesc() *PodDesc {
	return p.desc
}

func GenerateNetworkResultByNic(index int, nic PodNic, defaultGw bool) (*current.Result, error) {
	result := &current.Result{}
	ctrIf, ipConfigs, routes, err := getNetworkConfig(index, nic, defaultGw)
	if err != nil {
		return nil, errors.Wrap(err, "getNetworkConfig")
	}
	result.Interfaces = []*current.Interface{ctrIf}
	result.IPs = ipConfigs
	result.Routes = routes
	return result, nil
}

func GenerateNetworkResultByNics(nics []PodNic) (*current.Result, error) {
	result := &current.Result{}
	ipConfs := make([]*current.IPConfig, 0)
	ifs := make([]*current.Interface, 0)
	ifRoutes := make([]*types.Route, 0)
	for idx, nic := range nics {
		defaultGw := false
		if idx == 0 {
			defaultGw = true
		}
		ctrIf, ipConfigs, routes, err := getNetworkConfig(idx, nic, defaultGw)
		if err != nil {
			return nil, errors.Wrap(err, "getNetworkConfig")
		}
		ipConfs = append(ipConfs, ipConfigs...)
		ifs = append(ifs, ctrIf)
		ifRoutes = append(ifRoutes, routes...)
	}
	result.Interfaces = ifs
	result.IPs = ipConfs
	result.Routes = ifRoutes
	return result, nil
}

func getIPNet(ip string, mask int) (*net.IPNet, error) {
	cidr := fmt.Sprintf("%s/%d", ip, mask)
	ipAddr, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, errors.Wrapf(err, "ParseCIDR: %s", cidr)
	}
	return &net.IPNet{
		IP:   ipAddr,
		Mask: ipNet.Mask,
	}, nil
}

func getNetworkConfig(idx int, nic PodNic, defaultGw bool) (*current.Interface, []*current.IPConfig, []*types.Route, error) {
	ipConfigs := make([]*current.IPConfig, 0)
	ipn, err := getIPNet(nic.Ip, nic.Masklen)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "get ip network")
	}
	gatewayIp := net.ParseIP(nic.Gateway)
	ipConfig := &current.IPConfig{
		Version:   "4",
		Interface: &idx,
		Address:   *ipn,
		Gateway:   gatewayIp,
	}
	ipConfigs = append(ipConfigs, ipConfig)
	ctrIf := &current.Interface{
		Name: nic.Interface,
		Mac:  nic.Mac,
	}
	routes := make([]*types.Route, 0)
	if defaultGw {
		_, defaultNet, _ := net.ParseCIDR("0.0.0.0/0")
		defaultGateway := ipConfigs[0].Gateway
		route := &types.Route{
			Dst: *defaultNet,
			GW:  defaultGateway,
		}
		routes = append(routes, route)

	}
	// process ipv6
	if nic.Ip6 != "" {
		ip6n, err := getIPNet(nic.Ip6, nic.Masklen6)
		if err != nil {
			return nil, nil, nil, errors.Wrap(err, "get ipv6 network")
		}
		gatewayIp6 := net.ParseIP(nic.Gateway6)
		ip6Config := &current.IPConfig{
			Version:   "6",
			Interface: &idx,
			Address:   *ip6n,
			Gateway:   gatewayIp6,
		}
		ipConfigs = append(ipConfigs, ip6Config)

		// add ipv6 default route if ipv6 is configured
		if defaultGw {
			_, defaultNet6, _ := net.ParseCIDR("::/0")
			defaultGateway6 := net.ParseIP(nic.Gateway6)
			route6 := &types.Route{
				Dst: *defaultNet6,
				GW:  defaultGateway6,
			}
			routes = append(routes, route6)
		}
	}

	return ctrIf, ipConfigs, routes, nil
}

func setupNic(index int, nic PodNic, netns ns.NetNS, result *current.Result) error {
	// Create OVS client
	ovsCli, err := NewOVSClient()
	if err != nil {
		return errors.Wrap(err, "NewOVSClient")
	}

	ctrIfname := nic.GetInterface(index)
	//hostInterface, ctrInterface, err := setupVeth(ovsCli, index, netns, nic.Bridge)
	_, _, err = setupVeth(ovsCli, index, nic, netns)
	if err != nil {
		return errors.Wrap(err, "setupVeth")
	}

	// Configure the container hardware address and IP address(es)
	if err := netns.Do(func(_ ns.NetNS) error {
		_, err := net.InterfaceByName(ctrIfname)
		if err != nil {
			return errors.Wrapf(err, "net.InterfaceByName %q", ctrIfname)
		}

		// Add the IP to the interface
		if err := ConfigureIface(ctrIfname, result.IPs, result.Routes); err != nil {
			return errors.Wrap(err, "ConfigureIface")
		}
		return nil
	}); err != nil {
		return errors.Wrap(err, "Configure the container hardware address and IP address(es)")
	}
	return nil
}

func setupVeth(
	cli OVSClient,
	index int, nic PodNic, netns ns.NetNS) (*current.Interface, *current.Interface, error) {
	ctrIf := &current.Interface{}
	hostIf := &current.Interface{}

	hostIfname := nic.Ifname
	ctrIfname := nic.GetInterface(index)
	if err := ensureVethDeleted(hostIfname); err != nil {
		return nil, nil, errors.Wrapf(err, "ensure veth %q deleted", hostIfname)
	}

	if err := netns.Do(func(hostNS ns.NetNS) error {
		// create the veth pair in the container and move host endside into host netns
		hostVeth, ctrVeth, err := setupYunionVeth(index, hostIfname, ctrIfname, nic.Mac, nic.Mtu, hostNS)
		if err != nil {
			return errors.Wrap(err, "setupYunionVeth")
		}
		//log.Infof("makeVethPair hostVeth: %#v, containerVeth: %#v", hostVeth, ctrVeth)

		ctrIf.Name = ctrVeth.Name
		ctrIf.Mac = ctrVeth.HardwareAddr.String()
		ctrIf.Sandbox = netns.Path()
		hostIf.Name = hostVeth.Name

		// ip link set lo up
		if err := setLinkUp("lo"); err != nil {
			return errors.Wrap(err, "set loopback nic up")
		}
		return nil
	}); err != nil {
		return nil, nil, errors.Wrap(err, "netns.Do")
	}

	// need to lookup hostVeth again as its index has changed during ns move
	hostVeth, err := netlink.LinkByName(hostIf.Name)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to lookup host veth: %q", hostIf.Name)
	}
	hostIf.Mac = hostVeth.Attrs().HardwareAddr.String()

	if err := cli.AddPort(nic.Bridge, hostIf.Name, nic.Vlan); err != nil {
		return nil, nil, errors.Wrapf(err, "Add port to OVS: %s -> %s", hostIf.Name, nic.Bridge)
	}
	if nic.Vpc != nil && nic.Vpc.Provider == POD_NIC_PROVIDER_OVN {
		if err := cli.SetIfaceId(nic.NetId, hostIf.Name); err != nil {
			return nil, nil, errors.Wrapf(err, "Set interface id: %s -> %s", hostIf.Name, nic.Bridge)
		}
	}
	//log.Infof("Port %q added to %q", hostIf.Name, nic.Bridge)
	return hostIf, ctrIf, nil
}

func ensureVethDeleted(name string) error {
	// clean up if peer veth exists
	oldPeerVethName, err := netlink.LinkByName(name)
	if err == nil {
		if err := netlink.LinkDel(oldPeerVethName); err != nil {
			return errors.Wrapf(err, "failed to delete old peer veth %q", name)
		}
	}
	if err != nil {
		//log.Warningf("delete %q peer veth err: %v", name, err)
	}
	return nil
}

func setupYunionVeth(index int, hostVethName string, ctrIfName string, ctrMac string, mtu int, hostNS ns.NetNS) (net.Interface, net.Interface, error) {
	ctrVeth, err := makeVethPair(index, ctrIfName, hostVethName, mtu)
	if err != nil {
		return net.Interface{}, net.Interface{}, errors.Wrap(err, "makeVethPair")
	}

	if mac, err := net.ParseMAC(ctrMac); err != nil {
		return net.Interface{}, net.Interface{}, errors.Wrapf(err, "ParseMAC: %q", ctrMac)
	} else {
		if err := netlink.LinkSetHardwareAddr(ctrVeth, mac); err != nil {
			return net.Interface{}, net.Interface{}, errors.Wrapf(err, "netlink.LinkSetHardwareAddr: %q", mac.String())
		}
	}

	if err := netlink.LinkSetUp(ctrVeth); err != nil {
		return net.Interface{}, net.Interface{}, errors.Wrapf(err, "netlink.LinkSetup: %q", ctrVeth.Type())
	}

	hostVeth, err := netlink.LinkByName(hostVethName)
	if err != nil {
		return net.Interface{}, net.Interface{}, errors.Wrapf(err, "failed to lookup host veth: %q", hostVethName)
	}
	if err := netlink.LinkSetNsFd(hostVeth, int(hostNS.Fd())); err != nil {
		return net.Interface{}, net.Interface{}, errors.Wrapf(err, "failed to move veth %q to host netns %#v", hostVeth, hostNS)
	}

	if err := hostNS.Do(func(_ ns.NetNS) error {
		hostVeth, err := netlink.LinkByName(hostVethName)
		if err != nil {
			return errors.Wrapf(err, "failed to lookup host veth after moved: %q", hostVethName)
		}
		if err := netlink.LinkSetUp(hostVeth); err != nil {
			return errors.Wrapf(err, "failed to set host veth up after moved: %q", hostVethName)
		}
		return nil
	}); err != nil {
		return net.Interface{}, net.Interface{}, errors.Wrapf(err, "set link up at hostNS: %s", hostVethName)
	}

	return ifaceFromNetlinkLink(hostVeth), ifaceFromNetlinkLink(ctrVeth), nil
}

func ifaceFromNetlinkLink(l netlink.Link) net.Interface {
	a := l.Attrs()
	return net.Interface{
		Index:        a.Index,
		MTU:          a.MTU,
		Name:         a.Name,
		HardwareAddr: a.HardwareAddr,
		Flags:        a.Flags,
	}
}

func makeVethPair(index int, peerVethName, hostVethName string, mtu int) (netlink.Link, error) {
	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name:  peerVethName,
			Flags: net.FlagUp,
			MTU:   mtu,
		},
		PeerName: hostVethName,
	}
	if err := netlink.LinkAdd(veth); err != nil {
		return nil, errors.Wrap(err, "netlink.LinkAdd")
	}
	return veth, nil
}

func setLinkUp(name string) error {
	iface, err := netlink.LinkByName(name)
	if err != nil {
		return err
	}
	return netlink.LinkSetUp(iface)
}

func ConfigureIface(ifName string, ipConfigs []*current.IPConfig, routes []*types.Route) error {
	link, err := netlink.LinkByName(ifName)
	if err != nil {
		return errors.Wrap(err, "LinkByName")
	}

	if err := netlink.LinkSetUp(link); err != nil {
		return errors.Wrap(err, "LinkSetUp")
	}

	var v4gw, v6gw net.IP
	for _, ipc := range ipConfigs {
		addr := &netlink.Addr{IPNet: &ipc.Address, Label: ""}
		if err = netlink.AddrAdd(link, addr); err != nil {
			return fmt.Errorf("failed to add IP addr %v to %q: %v", ipc, ifName, err)
		}

		gwIsV4 := ipc.Gateway.To4() != nil
		if gwIsV4 && v4gw == nil {
			v4gw = ipc.Gateway
		} else if !gwIsV4 && v6gw == nil {
			v6gw = ipc.Gateway
		}
	}

	ip.SettleAddresses(ifName, 10)

	for _, r := range routes {
		routeIsV4 := r.Dst.IP.To4() != nil
		gw := r.GW
		if gw == nil {
			if routeIsV4 && v4gw != nil {
				gw = v4gw
			} else if !routeIsV4 && v6gw != nil {
				gw = v6gw
			}
		}
		if err = ip.AddRoute(&r.Dst, gw, link); err != nil {
			// we skip over duplicate routes as we assume the first one wins
			if !os.IsExist(err) {
				return fmt.Errorf("failed to add route '%v via %v dev %v': %v", r.Dst, gw, ifName, err)
			}
		}
	}

	return nil

}
