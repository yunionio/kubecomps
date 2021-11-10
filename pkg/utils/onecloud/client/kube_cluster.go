package client

import (
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

type CloudKubeClusterHelper struct {
	*ResourceHelper
}

func NewCloudKubeClusterHelper(s *mcclient.ClientSession) *CloudKubeClusterHelper {
	return &CloudKubeClusterHelper{
		ResourceHelper: NewResourceHelper(s, &modules.KubeClusters),
	}
}

func (h *CloudKubeClusterHelper) KubeClusters() modulebase.Manager {
	return h.ResourceHelper.Manager
}

type KubeClusterDetails struct {
	api.KubeClusterDetails
	Id        string `json:"id"`
	ManagerId string `json:"manager_id"`
}

func (h *CloudKubeClusterHelper) GetDetails(id string) (*KubeClusterDetails, error) {
	obj, err := h.KubeClusters().Get(h.session, id, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "get cloudregion by id %q", id)
	}
	out := new(KubeClusterDetails)
	if err := obj.Unmarshal(out); err != nil {
		return nil, errors.Wrap(err, "unmarshal json")
	}

	return out, nil
}
