package clusters

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/clusters/imported"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
)

func init() {
	sysDriver := NewDefaultSystemImportDriver()
	sysDriver.registerDriver(imported.NewExternalK8s())
	models.RegisterClusterDriver(sysDriver)
}

type SSystemImportDriver struct {
	*SExternalImportDriver
}

func NewDefaultSystemImportDriver() *SSystemImportDriver {
	drv := &SSystemImportDriver{
		SExternalImportDriver: NewExternalImportDriver(),
	}
	drv.providerType = api.ProviderTypeSystem
	drv.clusterResourceType = api.ClusterResourceTypeHost
	return drv
}

func (d *SSystemImportDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, createData *api.ClusterCreateInput) error {
	apiServer := createData.ImportData.ApiServer
	if apiServer == "" {
		return httperrors.NewInputParameterError("ApiServer must provide")
	}
	if err := d.SExternalImportDriver.ValidateCreateData(ctx, userCred, ownerId, query, createData); err != nil {
		return errors.Wrap(err, "SExternalImportDriver.ValidateCreateData")
	}
	return nil
}

func (d *SSystemImportDriver) ValidateDeleteCondition() error {
	return httperrors.NewNotAcceptableError("system cluster not allow delete")
}
