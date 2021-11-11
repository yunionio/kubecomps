package client

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

type ZoneHelper struct {
	*ResourceHelper
}

func NewZoneHelper(s *mcclient.ClientSession) *ZoneHelper {
	return &ZoneHelper{
		ResourceHelper: NewResourceHelper(s, &modules.Zones),
	}
}

func (h *ZoneHelper) Zones() modulebase.Manager {
	return h.ResourceHelper.Manager.(modulebase.Manager)
}

func (h *ZoneHelper) GetDetails(id string) (*api.ZoneDetails, error) {
	out := new(api.ZoneDetails)
	if err := h.ResourceHelper.GetDetails(id, out); err != nil {
		return nil, err
	}
	return out, nil
}
