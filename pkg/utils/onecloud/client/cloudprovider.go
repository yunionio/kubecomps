package client

import (
	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/pkg/errors"
)

type CloudproviderHelper struct {
	*ResourceHelper
}

func NewCloudproviderHelper(s *mcclient.ClientSession) *CloudproviderHelper {
	return &CloudproviderHelper{
		ResourceHelper: NewResourceHelper(s, &modules.Cloudproviders),
	}
}

func (h *CloudproviderHelper) Cloudproviders() modulebase.Manager {
	return h.ResourceHelper.Manager
}

func (h *CloudproviderHelper) GetDetails(id string) (*api.CloudproviderDetails, error) {
	obj, err := h.Cloudproviders().Get(h.session, id, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "get cloudprovider by id %q", id)
	}
	out := new(api.CloudproviderDetails)
	if err := obj.Unmarshal(out); err != nil {
		return nil, errors.Wrap(err, "unmarshal json")
	}

	return out, nil
}

func (h *CloudproviderHelper) GetCliRC(id string) (*jsonutils.JSONDict, error) {
	ret, err := h.Cloudproviders().GetSpecific(h.session, id, "clirc", jsonutils.NewDict())
	if err != nil {
		return nil, err
	}
	return ret.(*jsonutils.JSONDict), nil
}
