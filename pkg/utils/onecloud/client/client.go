package client

import (
	"yunion.io/x/onecloud/pkg/mcclient"
)

type IClient interface {
	Cloudregions() *CloudregionHelper
	Vpcs() *VpcHelper
	Servers() *ServerHelper
	GetCloudSSHPrivateKey() (string, error)
}

type sClient struct {
	s *mcclient.ClientSession
}

func NewClientSets(s *mcclient.ClientSession) IClient {
	cli := &sClient{
		s: s,
	}
	return cli
}

func (c *sClient) Cloudregions() *CloudregionHelper {
	return NewCloudregionHelper(c.s)
}

func (c *sClient) Vpcs() *VpcHelper {
	return NewVpcHelper(c.s)
}

func (c *sClient) Servers() *ServerHelper {
	return NewServerHelper(c.s)
}

func (c *sClient) GetCloudSSHPrivateKey() (string, error) {
	return GetCloudSSHPrivateKey(c.s)
}
