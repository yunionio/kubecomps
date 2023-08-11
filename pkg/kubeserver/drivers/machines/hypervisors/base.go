package hypervisors

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
)

type baseHypervisor struct {
	providerType api.ProviderType
}

func newBaseHypervisor(pt api.ProviderType) *baseHypervisor {
	return &baseHypervisor{providerType: pt}
}

func (b *baseHypervisor) GetHypervisor() api.ProviderType {
	return b.providerType
}

func (b *baseHypervisor) FindSystemDiskImage(s *mcclient.ClientSession, zoneId string) (jsonutils.JSONObject, error) {
	//TODO implement me
	panic("implement me")
}

func (b *baseHypervisor) PostPrepareServerResource(ctx context.Context, s *mcclient.ClientSession, m *models.SMachine, srv *compute.ServerDetails) error {
	return nil
}
