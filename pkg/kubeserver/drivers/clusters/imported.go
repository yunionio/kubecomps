package clusters

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	"yunion.io/x/kubecomps/pkg/utils/k8serrors"
)

func init() {
	importDriver := NewDefaultImportDriver()
	importDriver.drivers = drivers.NewDriverManager("")
	importDriver.registerDriver(
		newImportK8sDriver(),
		newImportOpenshiftDriver(),
	)
	models.RegisterClusterDriver(importDriver)
}

type iImportDriver interface {
	GetDistribution() string
	ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.ClusterCreateInput, config *rest.Config) error
	// GetClusterUsers query users resource from remote k8s cluster
	GetClusterUsers(cluster *models.SCluster, restCfg *rest.Config) ([]api.ClusterUser, error)
	// GetClusterUserGroups query groups resource from remote k8s cluster
	GetClusterUserGroups(cluster *models.SCluster, restCfg *rest.Config) ([]api.ClusterUserGroup, error)
}

type SDefaultImportDriver struct {
	*SBaseDriver
	drivers *drivers.DriverManager
}

func NewDefaultImportDriver() *SDefaultImportDriver {
	return &SDefaultImportDriver{
		SBaseDriver: newBaseDriver(api.ModeTypeImport, api.ProviderTypeExternal, api.ClusterResourceTypeUnknown),
	}
}

func (d *SDefaultImportDriver) registerDriver(drvs ...iImportDriver) {
	for idx := range drvs {
		if err := d.drivers.Register(drvs[idx], drvs[idx].GetDistribution()); err != nil {
			panic(fmt.Sprintf("import driver %s already registerd", drvs[idx].GetDistribution()))
		}
	}
}

func (d *SDefaultImportDriver) getDriver(distro string) iImportDriver {
	drv, err := d.drivers.Get(distro)
	if err != nil {
		panic(fmt.Errorf("Get driver %s: %v", distro, err))
	}
	return drv.(iImportDriver)
}

func (d *SDefaultImportDriver) getRegisterDistros() sets.String {
	ret := make([]string, 0)
	d.drivers.Range(func(key, val interface{}) bool {
		ret = append(ret, key.(string))
		return true
	})
	return sets.NewString(ret...)
}

func (d *SDefaultImportDriver) GetK8sVersions() []string {
	return []string{}
}

func (d *SDefaultImportDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, createData *api.ClusterCreateInput) error {
	// test kubeconfig is work
	importData := createData.ImportData
	if importData == nil {
		return httperrors.NewInputParameterError("not found import data")
	}
	if importData.Distribution == "" {
		importData.Distribution = api.ImportClusterDistributionK8s
	}
	if !d.getRegisterDistros().Has(importData.Distribution) {
		return httperrors.NewNotSupportedError("Not support import distribution %s", importData.Distribution)
	}
	apiServer := importData.ApiServer
	kubeconfig := importData.Kubeconfig
	restConfig, rawConfig, err := client.BuildClientConfig(apiServer, kubeconfig)
	if err != nil {
		return httperrors.NewNotAcceptableError("Invalid imported kubeconfig: %v", err)
	}
	newKubeconfig, err := runtime.Encode(clientcmdlatest.Codec, rawConfig)
	if err != nil {
		return httperrors.NewNotAcceptableError("Load kubeconfig error: %v", err)
	}
	importData.Kubeconfig = string(newKubeconfig)
	importData.ApiServer = restConfig.Host
	cli, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return k8serrors.NewGeneralError(err)
	}
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
	if sysCluster == nil {
		return httperrors.NewNotFoundError("Not found system cluster %v", sysCluster)
	}
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

	drv := d.getDriver(createData.ImportData.Distribution)
	if err := drv.ValidateCreateData(ctx, userCred, ownerId, createData, restConfig); err != nil {
		return errors.Wrapf(err, "check distribution %s", importData.Distribution)
	}
	createData.ImportData = importData

	return nil
}

func (d *SDefaultImportDriver) NeedGenerateCertificate() bool {
	return false
}

func (d *SDefaultImportDriver) NeedCreateMachines() bool {
	return false
}

func (d *SDefaultImportDriver) GetKubeconfig(cluster *models.SCluster) (string, error) {
	return cluster.Kubeconfig, nil
}

func (d *SDefaultImportDriver) ValidateCreateMachines(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	c *models.SCluster,
	_ *api.ClusterMachineCommonInfo,
	imageRepo *api.ImageRepository,
	data []*api.CreateMachineData) error {
	return httperrors.NewBadRequestError("Not support add machines")
}

func (d *SDefaultImportDriver) GetClusterUsers(cluster *models.SCluster, config *rest.Config) ([]api.ClusterUser, error) {
	return d.getDriver(cluster.Distribution).GetClusterUsers(cluster, config)
}

func (d *SDefaultImportDriver) GetClusterUserGroups(cluster *models.SCluster, config *rest.Config) ([]api.ClusterUserGroup, error) {
	return d.getDriver(cluster.Distribution).GetClusterUserGroups(cluster, config)
}
