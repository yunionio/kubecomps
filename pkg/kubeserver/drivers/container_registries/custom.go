package container_registries

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/container_registries/client"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
)

func init() {
	models.RegisterContainerRegistryDriver(newCustomImpl())
}

func newCustomImpl() models.IContainerRegistryDriver {
	return new(customImpl)
}

type customImpl struct{}

func (c customImpl) GetType() api.ContainerRegistryType {
	return api.ContainerRegistryTypeCustom
}

func (c customImpl) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.ContainerRegistryCreateInput) (*api.ContainerRegistryCreateInput, error) {
	return data, nil
}

func (c customImpl) CreateCredential(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.ContainerRegistryCreateInput) (string, error) {
	config := data.Config.Custom
	if config == nil {
		return "", nil
	}
	return createContainerImageSecret(ctx, userCred, ownerId, data.Name, &config.ContainerRegistryConfigCommon)
}

func (c customImpl) PreparePushImage(ctx context.Context, url string, conf *api.ContainerRegistryConfig, meta *client.ImageMetadata) error {
	return httperrors.NewNotSupportedError("custom not support prepare push image")
}

func (c customImpl) DownloadImage(ctx context.Context, url string, conf *api.ContainerRegistryConfig, input api.ContainerRegistryDownloadImageInput) (string, error) {
	return "", httperrors.NewNotSupportedError("custom not support download image")
}

func (c customImpl) GetDockerRegistryClient(url string, config *api.ContainerRegistryConfig) (client.Client, error) {
	return client.NewCustomClient(), nil
}
