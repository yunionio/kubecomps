package client

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type VpcHelper struct {
	*ResourceHelper
}

func NewVpcHelper(s *mcclient.ClientSession) *VpcHelper {
	return &VpcHelper{
		ResourceHelper: NewResourceHelper(s, &modules.Vpcs),
	}
}

func (h *VpcHelper) Vpcs() modulebase.Manager {
	return h.ResourceHelper.Manager
}

func (h *VpcHelper) GetDetails(id string) (*api.VpcDetails, error) {
	out := new(api.VpcDetails)
	if err := h.ResourceHelper.GetDetails(id, out); err != nil {
		return nil, err
	}

	return out, nil
}
