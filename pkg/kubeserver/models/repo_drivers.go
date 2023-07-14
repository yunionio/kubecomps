package models

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers"
)

var repoDrivers *drivers.DriverManager

func init() {
	repoDrivers = drivers.NewDriverManager("")
}

type IRepoDriver interface {
	GetBackend() api.RepoBackend

	ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.RepoCreateInput) (*api.RepoCreateInput, error)

	UploadChart(ctx context.Context, userCred mcclient.TokenCredential, repo *SRepo, chartName, chartPath string) (jsonutils.JSONObject, error)
}

func RegisterRepoDriver(driver IRepoDriver) {
	if err := repoDrivers.Register(driver, string(driver.GetBackend())); err != nil {
		panic(fmt.Sprintf("register helm repo driver backend %q: %v", driver.GetBackend(), err))
	}
}

func GetRepoDriver(backend api.RepoBackend) (IRepoDriver, error) {
	drv, err := repoDrivers.Get(string(backend))
	if err != nil {
		return nil, errors.Wrapf(err, "get helm repo driver by backend %q", backend)
	}
	return drv.(IRepoDriver), nil
}
