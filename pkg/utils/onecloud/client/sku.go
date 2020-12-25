package client

import (
	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type SkuHelper struct {
	*ResourceHelper
}

func NewSkuHelper(s *mcclient.ClientSession) *SkuHelper {
	return &SkuHelper{
		ResourceHelper: NewResourceHelper(s, &modules.ServerSkus),
	}
}

func (h *SkuHelper) Skus() *modules.ServerSkusManager {
	return h.ResourceHelper.Manager.(*modules.ServerSkusManager)
}

func (h *SkuHelper) GetDetails(name string, zoneId string) (*api.SServerSku, error) {
	input := map[string]interface{}{
		"search": name,
		"zone":   zoneId,
	}
	params := jsonutils.Marshal(input)
	ret, err := h.Skus().List(h.session, params)
	if err != nil {
		return nil, err
	}
	if len(ret.Data) == 0 {
		return nil, httperrors.NewNotFoundError("Not found sku by %s", params)
	}
	obj := ret.Data[0]
	out := new(api.SServerSku)
	if err := obj.Unmarshal(out); err != nil {
		return nil, err
	}
	return out, nil
}
