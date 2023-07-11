package helm_repos

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
)

func init() {
	models.RegisterRepoDriver(newNexusImpl())
}

func newNexusImpl() models.IRepoDriver {
	return new(nexusImpl)
}

type nexusImpl struct{}

func (n nexusImpl) GetBackend() api.RepoBackend {
	return api.RepoBackendNexus
}

func (n nexusImpl) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.RepoCreateInput) (*api.RepoCreateInput, error) {
	if data.Username == "" {
		return nil, httperrors.NewNotEmptyError("username is required")
	}
	if data.Password == "" {
		return nil, httperrors.NewNotEmptyError("password is required")
	}
	return data, nil
}

func (n nexusImpl) UploadChart(ctx context.Context, userCred mcclient.TokenCredential, repo *models.SRepo, chartName, chartPath string) (jsonutils.JSONObject, error) {
	fp, err := os.Open(chartPath)
	if err != nil {
		return nil, errors.Wrapf(err, "open %s file %q", chartName, chartPath)
	}
	uploadUrl := fmt.Sprintf("%s/%s", repo.Url, chartName)
	log.Infof("===uploadUrl: %s", uploadUrl)
	req, err := http.NewRequest(http.MethodPut, uploadUrl, fp)
	if err != nil {
		return nil, errors.Wrap(err, "new request")
	}
	req.SetBasicAuth(repo.Username, repo.Password)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "upload chart to nexus server")
	}
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "response from server after uploading chart to nexus")
	}
	log.Infof("response code: %d, status: %s, body: %s", resp.StatusCode, resp.Status, bodyText)
	if resp.StatusCode == http.StatusOK {
		return nil, nil
	}
	return nil, httperrors.NewJsonClientError(errors.Error("upload chart to nexus server"), fmt.Sprintf("status code: %s, body: %s", resp.Status, bodyText))
}
