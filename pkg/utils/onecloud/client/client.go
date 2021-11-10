package client

import (
	"yunion.io/x/onecloud/pkg/mcclient"
)

type IClient interface {
	Cloudregions() *CloudregionHelper
	Vpcs() *VpcHelper
	Zones() *ZoneHelper
	Networks() *NetworkHelper
	Servers() *ServerHelper
	Skus() *SkuHelper
	CloudKubeClusters() *CloudKubeClusterHelper
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

func (c *sClient) Zones() *ZoneHelper {
	return NewZoneHelper(c.s)
}

func (c *sClient) Networks() *NetworkHelper {
	return NewNetworkHelper(c.s)
}

func (c *sClient) Servers() *ServerHelper {
	return NewServerHelper(c.s)
}

func (c *sClient) Skus() *SkuHelper {
	return NewSkuHelper(c.s)
}

func (c *sClient) GetCloudSSHPrivateKey() (string, error) {
	return GetCloudSSHPrivateKey(c.s)
}

func (c *sClient) CloudKubeClusters() *CloudKubeClusterHelper {
	return NewCloudKubeClusterHelper(c.s)
}
