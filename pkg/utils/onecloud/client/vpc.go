package client

import (
	"yunion.io/x/pkg/errors"

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
	obj, err := h.Vpcs().Get(h.session, id, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "get vpc by id %q", id)
	}
	out := new(api.VpcDetails)
	if err := obj.Unmarshal(out); err != nil {
		return nil, errors.Wrap(err, "unmarshal json")
	}

	return out, nil
}
