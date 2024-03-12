package plugin

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"
	cniversion "github.com/containernetworking/cni/pkg/version"
	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/vishvananda/netlink"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/log/hooks"
	"yunion.io/x/pkg/errors"
)

func init() {
	// this ensures that main runs only on main thread (thread group leader).
	// since namespace ops (unshare, setns) are done for a single thread, we
	// must ensure that the goroutine does not jump from OS thread to thread
	runtime.LockOSThread()

	// init log hook
	initLog()
}

func initLog() {
	fileHook := &hooks.LogFileRotateHook{
		LogFileHook: hooks.LogFileHook{
			FileDir:  "/tmp/ocnet-log",
			FileName: "ocnet-cni.log",
		},
		RotateNum:  10,
		RotateSize: 1024,
	}
	if err := fileHook.Init(); err != nil {
		panic(fmt.Sprintf("fileHook.Init: %v", err))
	}
	log.Logger().AddHook(fileHook)
	log.DisableColors()
}

type NetConf struct {
	types.NetConf
}

func loadNetConf(bytes []byte, envArgs string) (*NetConf, string, error) {
	n := &NetConf{}
	if err := json.Unmarshal(bytes, n); err != nil {
		return nil, "", fmt.Errorf("failed to load netconf: %v", err)
	}
	return n, n.CNIVersion, nil
}

func cmdAdd(args *skel.CmdArgs) error {
	_, cniVersion, err := loadNetConf(args.StdinData, args.Args)
	if err != nil {
		return errors.Wrap(err, "loadNetConf")
	}
	pod, err := NewCloudPodFromCNIArgs(args.Args)
	if err != nil {
		return errors.Wrap(err, "NewCloudPodFromCNIArgs")
	}
	log.Infof("=====pod desc: %s", jsonutils.Marshal(pod.GetDesc()).PrettyString())

	netns, err := ns.GetNS(args.Netns)
	if err != nil {
		return errors.Wrapf(err, "failed to open netns %q", args.Netns)
	}
	defer netns.Close()

	nics := pod.GetDesc().Nics
	if len(nics) == 0 {
		return fmt.Errorf("Pod %s doesn't have nics", pod.Name)
	}

	result, err := GenerateNetworkResultByNics(nics)
	if err != nil {
		return errors.Wrap(err, "GenerateNetworkResultByNics")
	}

	for idx, nic := range nics {
		defaultGw := false
		if idx == 0 {
			defaultGw = true
		}
		nicResult, err := GenerateNetworkResultByNic(idx, nic, defaultGw)
		if err != nil {
			return errors.Wrapf(err, "GenerateNetworkResultByNic: %#v", nic)
		}
		if err := setupNic(idx, nic, netns, nicResult); err != nil {
			return errors.Wrap(err, "setupNic")
		}
	}

	log.Infof("====result: %s", jsonutils.Marshal(result).PrettyString())
	return types.PrintResult(result, cniVersion)
}

type linkNameToInterfaceReq struct {
	linkName string
	nsPath   string
}

func linkNamesToInterfaces(reqs []*linkNameToInterfaceReq) ([]*current.Interface, error) {
	resps := []*current.Interface{}
	for _, req := range reqs {
		resp, err := linkNameToInterface(req.linkName, req.nsPath)
		if err != nil {
			return nil, errors.Wrapf(err, "linkNameToInterface(%s, %s)", req.linkName, req.nsPath)
		}
		resps = append(resps, resp)
	}
	return resps, nil
}

func linkNameToInterface(ifName, nsPath string) (*current.Interface, error) {
	var err error
	var namespace ns.NetNS
	if nsPath == "" {
		if namespace, err = ns.GetCurrentNS(); err != nil {
			return nil, errors.Wrap(err, "GetCurrentNS")
		}
	} else {
		if namespace, err = ns.GetNS(nsPath); err != nil {
			return nil, errors.Wrap(err, "GetNS")
		}
	}
	defer namespace.Close()

	var if_ *current.Interface
	if err = namespace.Do(func(_ ns.NetNS) error {
		link, err := netlink.LinkByName(ifName)
		if err != nil {
			return errors.Wrapf(err, "netlink.LinkByName %s", ifName)
		}
		if_ = &current.Interface{
			Name:    ifName,
			Mac:     link.Attrs().HardwareAddr.String(),
			Sandbox: nsPath,
		}
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "namespace.Do")
	}
	return if_, nil
}

func cmdDel(args *skel.CmdArgs) error {
	_, _, err := loadNetConf(args.StdinData, args.Args)
	if err != nil {
		return errors.Wrap(err, "loadNetConf")
	}
	pod, err := NewCloudPodFromCNIArgs(args.Args)
	if err != nil {
		return errors.Wrap(err, "NewCloudPodFromCNIArgs")
	}

	ovsCli, err := NewOVSClient()
	if err != nil {
		return errors.Wrap(err, "NewOVSClient")
	}

	nics := pod.GetDesc().Nics
	for idx, nic := range nics {
		ovsPort := nic.Ifname
		reqs := []*linkNameToInterfaceReq{
			{
				linkName: nic.Bridge,
			},
			{
				linkName: ovsPort,
			},
		}
		if args.Netns != "" {
			reqs = append(reqs, &linkNameToInterfaceReq{
				linkName: nic.GetInterface(idx),
				nsPath:   args.Netns,
			})
		}
		if _, err := linkNamesToInterfaces(reqs); err != nil {
			return errors.Wrap(err, "failed to convert link name to ifname")
		}
		log.Infof("deleting port %s of Bridge %s", ovsPort, nic.Bridge)
		if err := ovsCli.DeletePort(nic.Bridge, ovsPort); err != nil {
			return errors.Wrapf(err, "delete port %s of %s", ovsPort, nic.Bridge)
		}
	}

	if args.Netns == "" {
		return nil
	}

	// There is a netns so try to clean up. Delete can be called multiple times
	// so don't return an error if the device is already removed.
	if err := ns.WithNetNSPath(args.Netns, func(_ ns.NetNS) error {
		for idx, nic := range nics {
			_, err := ip.DelLinkByNameAddr(nic.GetInterface(idx))
			if err != nil && err == ip.ErrLinkNotFound {
				continue
			}
			return errors.Wrapf(err, "ip.DelLinkByNameAddr(%s, %v)", nic.GetInterface(idx), netlink.FAMILY_ALL)
		}
		return nil
	}); err != nil {
		return errors.Wrapf(err, "ns.WithNetNSPath %s", args.Netns)
	}
	return nil
}

func cmdCheck(args *skel.CmdArgs) error {
	return nil
}

func Main(version string) {
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, cniversion.All, version)
}
