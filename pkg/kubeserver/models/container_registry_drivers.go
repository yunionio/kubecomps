package models

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/container_registries/client"
)

var containerRegistryDrivers *drivers.DriverManager

func init() {
	containerRegistryDrivers = drivers.NewDriverManager("")
}

type IContainerRegistryDriver interface {
	GetType() api.ContainerRegistryType

	GetDockerRegistryClient(url string, config *api.ContainerRegistryConfig) (client.Client, error)

	ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.ContainerRegistryCreateInput) (*api.ContainerRegistryCreateInput, error)

	PreparePushImage(ctx context.Context, url string, conf *api.ContainerRegistryConfig, meta *client.ImageMetadata) error
	DownloadImage(ctx context.Context, url string, conf *api.ContainerRegistryConfig, input api.ContainerRegistryDownloadImageInput) (string, error)
}

func RegisterContainerRegistryDriver(driver IContainerRegistryDriver) {
	if err := containerRegistryDrivers.Register(driver, string(driver.GetType())); err != nil {
		panic(fmt.Sprintf("register container registry driver type %q: %v", driver.GetType(), err))
	}
}

func GetContainerRegistryDriver(rType api.ContainerRegistryType) (IContainerRegistryDriver, error) {
	drv, err := containerRegistryDrivers.Get(string(rType))
	if err != nil {
		return nil, errors.Wrapf(err, "get container registry driver by type %q", rType)
	}
	return drv.(IContainerRegistryDriver), nil
}
