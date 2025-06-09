package plugin

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/sdnagent/pkg/agent"
	pb "yunion.io/x/sdnagent/pkg/agent/proto"
)

type OVSClient interface {
	AddPort(bridge, port string, vlanId int) error
	DeletePort(bridge, port string) error
	SetIfaceId(id string, name string) error
}

func NewOVSClient() (OVSClient, error) {
	cli, err := agent.NewClient("/var/run/onecloud/sdnagent.sock")
	if err != nil {
		return nil, errors.Wrap(err, "new yunion sdnagent client")
	}
	sw := &ovsClient{
		agentCli: cli,
	}
	return sw, nil
}

type ovsClient struct {
	agentCli *agent.AgentClient
}

func newTimeoutCtx() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(10*time.Second))
	return ctx
}

func (sw *ovsClient) AddPort(bridge, port string, vlanId int) error {
	/*return sw.agentCli.W(sw.agentCli.VSwitch.AddBridgePort(newTimeoutCtx(), &pb.AddBridgePortRequest{
		Bridge: bridge,
		Port:   port,
	}))*/
	args := []string{"--may-exist", "add-port", bridge, port}
	if vlanId > 1 {
		args = append(args, fmt.Sprintf("tag=%d", vlanId))
	}
	ovsCmd := exec.Command("ovs-vsctl", args...)
	if out, err := ovsCmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "add port to ovs %v: %s", args, out)
	}
	return nil
}

func (sw *ovsClient) SetIfaceId(netId string, ifName string) error {
	ovsCmd := exec.Command("ovs-vsctl", "set", "Interface", ifName, fmt.Sprintf("external-ids:iface-id=iface-%s-%s", netId, ifName))
	if out, err := ovsCmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "set external ids: %s", out)
	}
	return nil
}

func (sw *ovsClient) DeletePort(bridge, port string) error {
	return sw.agentCli.W(sw.agentCli.VSwitch.DelBridgePort(newTimeoutCtx(), &pb.DelBridgePortRequest{
		Bridge: bridge,
		Port:   port,
	}))
}
