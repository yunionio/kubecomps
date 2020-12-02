package clusters

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
)

type SDefaultSystemImportDriver struct {
	*SDefaultImportDriver
}

func NewDefaultSystemImportDriver() *SDefaultSystemImportDriver {
	drv := &SDefaultSystemImportDriver{
		SDefaultImportDriver: NewDefaultImportDriver(),
	}
	drv.providerType = api.ProviderTypeSystem
	drv.clusterResourceType = api.ClusterResourceTypeHost
	return drv
}

func init() {
	models.RegisterClusterDriver(NewDefaultSystemImportDriver())
}

func (d *SDefaultSystemImportDriver) GetK8sVersions() []string {
	return []string{}
}

func (d *SDefaultSystemImportDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, createData *api.ClusterCreateInput) error {
	apiServer := createData.ImportData.ApiServer
	if apiServer == "" {
		return httperrors.NewInputParameterError("ApiServer must provide")
	}
	kubeconfig := createData.ImportData.Kubeconfig
	cli, _, err := client.BuildClient(apiServer, kubeconfig)
	if err != nil {
		return httperrors.NewNotAcceptableError("Invalid imported kubeconfig: %v", err)
	}
	version, err := cli.Discovery().ServerVersion()
	if err != nil {
		return httperrors.NewGeneralError(errors.Wrap(err, "Get kubernetes version"))
	}
	createData.Version = version.String()
	return nil
}

func (d *SDefaultSystemImportDriver) NeedGenerateCertificate() bool {
	return false
}

func (d *SDefaultSystemImportDriver) NeedCreateMachines() bool {
	return false
}

func (d *SDefaultSystemImportDriver) GetKubeconfig(cluster *models.SCluster) (string, error) {
	return cluster.Kubeconfig, nil
}

func (d *SDefaultSystemImportDriver) ValidateDeleteCondition() error {
	return httperrors.NewNotAcceptableError("system cluster not allow delete")
}
