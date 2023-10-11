package clusters

import (
	"context"
	"fmt"

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
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/clusters/imported"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	"yunion.io/x/kubecomps/pkg/utils/k8serrors"
)

var (
	_ models.IClusterDriver = new(sImportBaseDriver)
	_ models.IClusterDriver = new(SCloudImportBaseDriver)
)

type sImportBaseDriver struct {
	*SBaseDriver
	drivers *drivers.DriverManager
}

func newImportBaseDriver(provider api.ProviderType, resType api.ClusterResourceType) *sImportBaseDriver {
	return &sImportBaseDriver{
		SBaseDriver: newBaseDriver(
			api.ModeTypeImport,
			provider,
			resType),
		drivers: drivers.NewDriverManager(""),
	}
}

func (d *sImportBaseDriver) registerDriver(drvs ...imported.IImportDriver) {
	for idx := range drvs {
		if err := d.drivers.Register(drvs[idx], drvs[idx].GetDistribution()); err != nil {
			panic(fmt.Sprintf("import driver %s already registerd", drvs[idx].GetDistribution()))
		}
	}
}

func (d *sImportBaseDriver) getDriver(distro string) imported.IImportDriver {
	drv, err := d.drivers.Get(distro)
	if err != nil {
		panic(fmt.Errorf("Get driver %s: %v", distro, err))
	}
	return drv.(imported.IImportDriver)
}

func (d *sImportBaseDriver) getRegisterDistros() sets.String {
	ret := make([]string, 0)
	d.drivers.Range(func(key, val interface{}) bool {
		ret = append(ret, key.(string))
		return true
	})
	return sets.NewString(ret...)
}

func (d *sImportBaseDriver) GetK8sVersions() []string {
	return []string{}
}

func (d *sImportBaseDriver) getKubeClientByConfig(apiServer string, kubeconfig string) ([]byte, *rest.Config, kubernetes.Interface, error) {
	restConfig, rawConfig, err := client.BuildClientConfig(apiServer, kubeconfig)
	if err != nil {
		return nil, nil, nil, httperrors.NewNotAcceptableError("Invalid imported kubeconfig: %v", err)
	}
	newKubeconfig, err := runtime.Encode(clientcmdlatest.Codec, rawConfig)
	if err != nil {
		return nil, nil, nil, httperrors.NewNotAcceptableError("Load kubeconfig error: %v", err)
	}
	cli, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, nil, k8serrors.NewGeneralError(err)
	}
	return newKubeconfig, restConfig, cli, nil
}

func (d *sImportBaseDriver) NeedGenerateCertificate() bool {
	return false
}

func (d *sImportBaseDriver) NeedCreateMachines() bool {
	return false
}

func (d *sImportBaseDriver) GetKubeconfig(cluster *models.SCluster) (string, error) {
	return cluster.Kubeconfig, nil
}

func (d *sImportBaseDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, createData *api.ClusterCreateInput) error {
	// test kubeconfig is work
	importData := createData.ImportData
	if importData == nil {
		return httperrors.NewInputParameterError("not found import data")
	}
	if importData.Distribution == "" {
		importData.Distribution = api.ImportClusterDistributionK8s
	}

	// check distribution
	dists := d.getRegisterDistros()
	if !dists.Has(importData.Distribution) {
		return httperrors.NewNotSupportedError("Not support import distribution %s in %v", importData.Distribution, dists)
	}

	apiServer := importData.ApiServer
	kubeconfig := importData.Kubeconfig
	newKubeconfig, restConfig, cli, err := d.getKubeClientByConfig(apiServer, kubeconfig)
	if err != nil {
		return httperrors.NewNotSupportedError("get kubernetes client by config: %v", err)
	}
	importData.Kubeconfig = string(newKubeconfig)
	importData.ApiServer = restConfig.Host
	version, err := cli.Discovery().ServerVersion()
	if err != nil {
		return k8serrors.NewGeneralError(err)
	}
	createData.Version = version.String()

	drv := d.getDriver(createData.ImportData.Distribution)
	if err := drv.ValidateCreateData(ctx, userCred, ownerId, createData, restConfig); err != nil {
		return errors.Wrapf(err, "check distribution %s", importData.Distribution)
	}

	createData.ImportData = importData

	return nil
}

func (d *sImportBaseDriver) ValidateCreateMachines(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	c *models.SCluster,
	_ *api.ClusterMachineCommonInfo,
	imageRepo *api.ImageRepository,
	data []*api.CreateMachineData) error {
	return httperrors.NewBadRequestError("Not support add machines")
}

func (d *sImportBaseDriver) GetClusterUsers(cluster *models.SCluster, config *rest.Config) ([]api.ClusterUser, error) {
	return d.getDriver(cluster.Distribution).GetClusterUsers(cluster, config)
}

func (d *sImportBaseDriver) GetClusterUserGroups(cluster *models.SCluster, config *rest.Config) ([]api.ClusterUserGroup, error) {
	return d.getDriver(cluster.Distribution).GetClusterUserGroups(cluster, config)
}

type SCloudImportBaseDriver struct {
	*sImportBaseDriver
}

func newCloudImportDriver(drv imported.ICloudImportDriver) *SCloudImportBaseDriver {
	cd := &SCloudImportBaseDriver{
		sImportBaseDriver: newImportBaseDriver(drv.GetProvider(), api.ClusterResourceTypeGuest),
	}
	cd.registerDriver(drv)
	return cd
}
