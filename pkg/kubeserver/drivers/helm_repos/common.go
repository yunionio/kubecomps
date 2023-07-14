package helm_repos

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
)

func init() {
	models.RegisterRepoDriver(newCommonImpl())
}

func newCommonImpl() models.IRepoDriver {
	return new(commonImpl)
}

type commonImpl struct{}

func (c commonImpl) GetBackend() api.RepoBackend {
	return api.RepoBackendCommon
}

func (c commonImpl) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.RepoCreateInput) (*api.RepoCreateInput, error) {
	return data, nil
}

func (c commonImpl) UploadChart(ctx context.Context, userCred mcclient.TokenCredential, repo *models.SRepo, chartName, chartPath string) (jsonutils.JSONObject, error) {
	return nil, httperrors.NewNotAcceptableError("common helm repository backend is not support upload chart")
}
