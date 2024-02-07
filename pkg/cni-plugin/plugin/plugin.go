package plugin

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/040"
	cniversion "github.com/containernetworking/cni/pkg/version"
	"github.com/containernetworking/plugins/pkg/ns"

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
		if err := setupNic(idx, nic, netns); err != nil {
			return errors.Wrap(err, "setupNic")
		}
	}

	return types.PrintResult(result, cniVersion)
}

func cmdDel(args *skel.CmdArgs) error {
	return nil
}

func cmdCheck(args *skel.CmdArgs) error {
	return nil
}

func Main(version string) {
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, cniversion.All, version)
}
