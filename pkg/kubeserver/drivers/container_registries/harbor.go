package container_registries

import (
	"context"
	"strings"

	"github.com/goharbor/go-client/pkg/harbor"
	"github.com/goharbor/go-client/pkg/sdk/v2.0/client/project"
	"github.com/goharbor/go-client/pkg/sdk/v2.0/client/repository"
	hbmodels "github.com/goharbor/go-client/pkg/sdk/v2.0/models"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/container_registries/client"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
)

func init() {
	models.RegisterContainerRegistryDriver(newHarborImpl())
}

func newHarborImpl() models.IContainerRegistryDriver {
	return new(harborImpl)
}

type harborImpl struct{}

func (h harborImpl) DownloadImage(ctx context.Context, url string, conf *api.ContainerRegistryConfig, input api.ContainerRegistryDownloadImageInput) (string, error) {
	return "", httperrors.NewNotSupportedError("harbor not support download image")
}

func (h harborImpl) GetType() api.ContainerRegistryType {
	return api.ContainerRegistryTypeHarbor
}

func (h harborImpl) GetDockerRegistryClient(url string, config *api.ContainerRegistryConfig) (client.Client, error) {
	if config.Harbor == nil {
		return nil, errors.Errorf("harbor config is nil")
	}
	return client.NewClient(url, client.DockerAuthConfig{
		Username: config.Harbor.Username,
		Password: config.Harbor.Password,
	})
}

func (h harborImpl) getClient(url string, config *api.ContainerRegistryConfigHarbor) (*harbor.ClientSet, error) {
	c := &harbor.ClientSetConfig{
		URL:      url,
		Insecure: true,
		Username: config.Username,
		Password: config.Password,
	}
	cs, err := harbor.NewClientSet(c)
	if err != nil {
		return nil, errors.Wrap(err, "new harbor clientset")
	}
	return cs, nil
}

func (h harborImpl) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.ContainerRegistryCreateInput) (*api.ContainerRegistryCreateInput, error) {
	config := data.Config.Harbor
	if config == nil {
		return nil, httperrors.NewInputParameterError("Configuration of harbor is nil")
	}

	cs, err := h.getClient(data.Url, config)
	if err != nil {
		return nil, errors.Wrap(err, "get harbor client")
	}

	params := repository.NewListAllRepositoriesParams()
	repos, err := cs.V2().Repository.ListAllRepositories(ctx, params)
	if err != nil {
		return nil, errors.Wrap(err, "ListAllRepositories")
	}
	log.Errorf("list harbor repos: %s", jsonutils.Marshal(repos).PrettyString())

	return data, nil
}
func (h harborImpl) PreparePushImage(ctx context.Context, url string, conf *api.ContainerRegistryConfig, meta *client.ImageMetadata) error {
	// get project name
	ref := meta.Ref
	parts := strings.Split(ref.Repository, "/")
	if len(parts) == 0 {
		return errors.Errorf("get project by %q", ref.Repository)
	}
	projectName := parts[0]
	cs, err := h.getClient(url, conf.Harbor)
	if err != nil {
		return errors.Wrap(err, "get harbor client")
	}
	trueObj := true
	params := project.NewCreateProjectParams()
	params.Project = &hbmodels.ProjectReq{
		ProjectName: projectName,
		Public:      &trueObj,
	}
	ret, err := cs.V2().Project.CreateProject(ctx, params)
	log.Infof("create project %q, ret: %#v, error: %v", projectName, ret, err)
	return nil
}
