package client

import (
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type CloudregionHelper struct {
	*ResourceHelper
}

func NewCloudregionHelper(s *mcclient.ClientSession) *CloudregionHelper {
	return &CloudregionHelper{
		ResourceHelper: NewResourceHelper(s, &modules.Cloudregions),
	}
}

func (h *CloudregionHelper) Cloudregions() modulebase.Manager {
	return h.ResourceHelper.Manager
}

func (h *CloudregionHelper) GetDetails(id string) (*api.CloudregionDetails, error) {
	obj, err := h.Cloudregions().Get(h.session, id, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "get cloudregion by id %q", id)
	}
	out := new(api.CloudregionDetails)
	if err := obj.Unmarshal(out); err != nil {
		return nil, errors.Wrap(err, "unmarshal json")
	}

	return out, nil
}
