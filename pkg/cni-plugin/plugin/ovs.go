package plugin

import (
	"context"
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/sdnagent/pkg/agent"
	pb "yunion.io/x/sdnagent/pkg/agent/proto"
)

type OVSClient interface {
	AddPort(bridge, port string) error
	DeletePort(bridge, port string) error
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

func (sw *ovsClient) AddPort(bridge, port string) error {
	return sw.agentCli.W(sw.agentCli.VSwitch.AddBridgePort(newTimeoutCtx(), &pb.AddBridgePortRequest{
		Bridge: bridge,
		Port:   port,
	}))
}

func (sw *ovsClient) DeletePort(bridge, port string) error {
	return sw.agentCli.W(sw.agentCli.VSwitch.DelBridgePort(newTimeoutCtx(), &pb.DelBridgePortRequest{
		Bridge: bridge,
		Port:   port,
	}))
}
