package client

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

type NetworkHelper struct {
	*ResourceHelper
}

func NewNetworkHelper(s *mcclient.ClientSession) *NetworkHelper {
	return &NetworkHelper{
		ResourceHelper: NewResourceHelper(s, &modules.Networks),
	}
}

func (h *NetworkHelper) Networks() modulebase.Manager {
	return h.ResourceHelper.Manager.(modulebase.Manager)
}

func (h *NetworkHelper) GetDetails(id string) (*api.NetworkDetails, error) {
	out := new(api.NetworkDetails)
	if err := h.ResourceHelper.GetDetails(id, out); err != nil {
		return nil, err
	}
	return out, nil
}
