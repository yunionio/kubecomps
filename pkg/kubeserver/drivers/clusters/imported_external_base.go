package clusters

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/clusters/imported"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	"yunion.io/x/kubecomps/pkg/utils/k8serrors"
)

func init() {
	externalDriver := NewExternalImportDriver()
	externalDriver.registerDriver(
		imported.NewExternalK8s(),
		imported.NewExternalOpenshift(),
	)
	models.RegisterClusterDriver(externalDriver)
}

type SExternalImportDriver struct {
	*sImportBaseDriver
}

func NewExternalImportDriver() *SExternalImportDriver {
	return &SExternalImportDriver{
		sImportBaseDriver: newImportBaseDriver(api.ProviderTypeExternal, api.ClusterResourceTypeUnknown),
	}
}

func (d *SExternalImportDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, createData *api.ClusterCreateInput) error {
	if err := d.sImportBaseDriver.ValidateCreateData(ctx, userCred, ownerId, query, createData); err != nil {
		return errors.Wrap(err, "sImportBaseDriver.ValidateCreateData")
	}

	importData := createData.ImportData
	newKubeconfig, _, cli, err := d.sImportBaseDriver.getKubeClientByConfig(importData.ApiServer, importData.Kubeconfig)
	if err != nil {
		return err
	}

	importData.Kubeconfig = string(newKubeconfig)
	version, err := cli.Discovery().ServerVersion()
	if err != nil {
		return k8serrors.NewGeneralError(err)
	}
	createData.Version = version.String()
	// check system cluster duplicate imported
	sysCluster, err := models.ClusterManager.GetSystemCluster()
	if err != nil {
		return httperrors.NewGeneralError(errors.Wrap(err, "Get system cluster %v"))
	}
	if sysCluster != nil {
		k8sSvc, err := cli.CoreV1().Services(metav1.NamespaceDefault).Get(context.Background(), "kubernetes", metav1.GetOptions{})
		if err != nil {
			return err
		}
		sysCli, err := sysCluster.GetK8sClient()
		if err != nil {
			return httperrors.NewGeneralError(errors.Wrap(err, "Get system cluster k8s client"))
		}
		sysK8SSvc, err := sysCli.CoreV1().Services(metav1.NamespaceDefault).Get(context.Background(), "kubernetes", metav1.GetOptions{})
		if err != nil {
			return err
		}
		if k8sSvc.UID == sysK8SSvc.UID {
			return httperrors.NewNotAcceptableError("cluster already imported as default system cluster")
		}
	}

	createData.ImportData = importData

	return nil
}
