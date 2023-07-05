package container_registries

import (
	"context"
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/container_registries/client"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
)

func init() {
	models.RegisterContainerRegistryDriver(newCommonImpl())
}

func newCommonImpl() models.IContainerRegistryDriver {
	return new(commonImpl)
}

type commonImpl struct{}

func (c commonImpl) GetType() api.ContainerRegistryType {
	return api.ContainerRegistryTypeCommon
}

func (c commonImpl) GetDockerRegistryClient(url string, config *api.ContainerRegistryConfig) (client.Client, error) {
	if config.Common == nil {
		return nil, errors.Errorf("harbor config is nil")
	}
	return client.NewClient(url, client.DockerAuthConfig{
		Username: config.Common.Username,
		Password: config.Common.Password,
	})
}

func (c commonImpl) GetPathPrefix(inputUrl string) (string, error) {
	regUrl, err := url.Parse(inputUrl)
	if err != nil {
		return "", httperrors.NewInputParameterError("Invalid url: %q", inputUrl)
	}
	if regUrl.Path == "" {
		return "", httperrors.NewInputParameterError("Url %q must with path prefix used for a repository namespace", inputUrl)
	}
	return regUrl.Path, nil
}

func (c commonImpl) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.ContainerRegistryCreateInput) (*api.ContainerRegistryCreateInput, error) {
	config := data.Config.Common
	if config == nil {
		return nil, httperrors.NewInputParameterError("Configuration of common is nil")
	}
	if _, err := c.GetPathPrefix(data.Url); err != nil {
		return nil, err
	}
	cli, err := c.GetDockerRegistryClient(data.Url, &data.Config)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Get docker registry client: %v", err)
	}
	if err := cli.Ping(ctx); err != nil {
		return nil, httperrors.NewInputParameterError("Ping docker registry %q: %v", data.Url, err)
	}
	return data, nil
}

func (c commonImpl) PreparePushImage(ctx context.Context, url string, conf *api.ContainerRegistryConfig, meta *client.ImageMetadata) error {
	parts := strings.Split(meta.Ref.Repository, "/")
	meta.Ref.Repository = parts[len(parts)-1]
	return nil
}
