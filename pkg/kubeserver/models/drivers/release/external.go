package release

import (
	"context"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
)

func init() {
	models.GetReleaseManager().RegisterDriver(newExternalDriver())
}

func newExternalDriver() models.IReleaseDriver {
	return new(externalDriver)
}

type externalDriver struct{}

func (d *externalDriver) GetType() api.RepoType {
	return api.RepoTypeExternal
}

func (d *externalDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, data *api.ReleaseCreateInput) (*api.ReleaseCreateInput, error) {
	cluster, err := models.ClusterManager.FetchClusterByIdOrName(ctx, userCred, data.ClusterId)
	if err != nil {
		return nil, err
	}
	data.ClusterId = cluster.GetId()
	_, err = client.GetManagerByCluster(cluster)
	if err != nil {
		return nil, err
	}
	if data.NamespaceId == "" {
		return nil, httperrors.NewNotEmptyError("namespace")
	}
	nInput, err := models.GetReleaseManager().SNamespaceResourceBaseManager.ValidateCreateData(ctx, userCred, ownerCred, nil, &data.NamespaceResourceCreateInput)
	if err != nil {
		return nil, err
	}
	data.NamespaceResourceCreateInput = *nInput
	return data, nil
}

func (d *externalDriver) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, release *models.SRelease, data *api.ReleaseCreateInput) error {
	release.ClusterId = data.ClusterId
	return nil
}
