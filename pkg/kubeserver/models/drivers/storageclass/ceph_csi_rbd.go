package storageclass

import (
	"context"
	"database/sql"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	"yunion.io/x/kubecomps/pkg/utils/ceph"
)

const (
	CSIStorageK8SIO = "csi.storage.k8s.io"
)

func init() {
	models.GetStorageClassManager().RegisterDriver(
		api.StorageClassProvisionerCephCSIRBD,
		newCephCSIRBD(),
	)
}

func GetCSIParamsKey(suffix string) string {
	return CSIStorageK8SIO + "/" + suffix
}

type CephCSIRBD struct{}

func newCephCSIRBD() models.IStorageClassDriver {
	return new(CephCSIRBD)
}

func (drv *CephCSIRBD) getUserKeyFromSecret(cliMan *client.ClusterManager, name, namespace string) (string, string, error) {
	cli := cliMan.GetClientset()
	secret, err := cli.CoreV1().Secrets(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return "", "", err
	} else if secret.Type != api.SecretTypeCephCSI {
		return "", "", httperrors.NewInputParameterError("%s/%s secret type is not %s", namespace, name, api.SecretTypeCephCSI)
	}
	uId := string(secret.Data["userID"])
	key := string(secret.Data["userKey"])
	if err != nil {
		return "", "", httperrors.NewNotAcceptableError("%s/%s user key decode error: %v", namespace, name, err)
	}
	return uId, key, nil
}

type cephConfig struct {
	api.ComponentCephCSIConfigCluster
	User string
	Key  string
}

func (drv *CephCSIRBD) getCephConfig(userCred mcclient.TokenCredential, cli *client.ClusterManager, data *api.StorageClassCreateInput) (*cephConfig, error) {
	input := data.CephCSIRBD
	if input == nil {
		return nil, httperrors.NewInputParameterError("cephCSIRBD config is empty")
	}
	cluster := cli.GetClusterObject().(*models.SCluster)
	secretName := input.SecretName
	if secretName == "" {
		return nil, httperrors.NewNotEmptyError("secretName is empty")
	}
	secretNamespace := input.SecretNamespace
	if secretNamespace == "" {
		return nil, httperrors.NewNotEmptyError("secretNamespace is empty")
	}
	nsObj, err := models.FetchClusterResourceByIdOrName(models.GetNamespaceManager(), userCred, cluster.GetId(), "", secretNamespace)
	if err != nil {
		return nil, errors.Wrapf(err, "get cluster %s namespace %s", cluster.GetId(), secretNamespace)
	}
	secObj, err := models.FetchClusterResourceByIdOrName(models.GetSecretManager(), userCred, cluster.GetId(), nsObj.GetId(), secretName)
	if err != nil {
		return nil, errors.Wrapf(err, "get cluster %s secret %s/%s object", cluster.GetId(), nsObj.GetName(), secretName)
	}

	user, key, err := drv.getUserKeyFromSecret(cli, secObj.GetName(), nsObj.GetName())
	if err != nil {
		return nil, err
	}

	input.SecretName = secObj.GetName()
	input.SecretNamespace = nsObj.GetName()

	// check clusterId
	component, err := cluster.GetComponentByType(api.ClusterComponentCephCSI)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, httperrors.NewNotFoundError("not found cluster %s component %s", cluster.GetName(), api.ClusterComponentCephCSI)
		}
		return nil, err
	}
	settings, err := component.GetSettings()
	if err != nil {
		return nil, err
	}
	if input.ClusterId == "" {
		return nil, httperrors.NewInputParameterError("clusterId is empty")
	}
	cephConf, err := drv.validateClusterId(input.ClusterId, settings.CephCSI)
	if err != nil {
		return nil, err
	}
	return &cephConfig{
		cephConf,
		user,
		key,
	}, nil
}

func (drv *CephCSIRBD) ValidateCreateData(userCred mcclient.TokenCredential, cli *client.ClusterManager, data *api.StorageClassCreateInput) (*api.StorageClassCreateInput, error) {
	cephConf, err := drv.getCephConfig(userCred, cli, data)
	if err != nil {
		return nil, err
	}

	input := data.CephCSIRBD

	if input.Pool == "" {
		return nil, httperrors.NewInputParameterError("pool is empty")
	}
	if err := drv.validatePool(cephConf.Monitors, cephConf.User, cephConf.Key, input.Pool); err != nil {
		return nil, err
	}

	if input.CSIFsType == "" {
		return nil, httperrors.NewInputParameterError("csiFsType is empty")
	} else {
		if !utils.IsInStringArray(input.CSIFsType, []string{"ext4", "xfs"}) {
			return nil, httperrors.NewInputParameterError("unsupport fsType %s", input.CSIFsType)
		}
	}

	if input.ImageFeatures != "layering" {
		return nil, httperrors.NewInputParameterError("imageFeatures only support 'layering' currently")
	}
	return data, nil
}

func (drv *CephCSIRBD) listPools(monitors []string, user string, key string) ([]string, error) {
	cephCli, err := ceph.NewClient(user, key, monitors...)
	if err != nil {
		return nil, errors.Wrap(err, "new ceph client")
	}
	return cephCli.ListPoolsNoDefault()
}

func (drv *CephCSIRBD) validateClusterId(cId string, conf *api.ComponentSettingCephCSI) (api.ComponentCephCSIConfigCluster, error) {
	for _, c := range conf.Config {
		if c.ClsuterId == cId {
			return c, nil
		}
	}
	return api.ComponentCephCSIConfigCluster{}, httperrors.NewNotFoundError("Not found clusterId %s in component config", cId)
}

func (drv *CephCSIRBD) validatePool(monitors []string, user string, key string, pool string) error {
	pools, err := drv.listPools(monitors, user, key)
	if err != nil {
		return err
	}
	if !utils.IsInStringArray(pool, pools) {
		return httperrors.NewNotFoundError("not found pool %s in %v", pool, monitors)
	}
	return nil
}

func (drv *CephCSIRBD) ConnectionTest(userCred mcclient.TokenCredential, cli *client.ClusterManager, data *api.StorageClassCreateInput) (*api.StorageClassTestResult, error) {
	cephConf, err := drv.getCephConfig(userCred, cli, data)
	if err != nil {
		return nil, err
	}
	pools, err := drv.listPools(cephConf.Monitors, cephConf.User, cephConf.Key)
	if err != nil {
		return nil, err
	}
	ret := new(api.StorageClassTestResult)
	ret.CephCSIRBD = &api.StorageClassTestResultCephCSIRBD{Pools: pools}
	return ret, nil
}

func (drv *CephCSIRBD) ToStorageClassParams(input *api.StorageClassCreateInput) (map[string]string, error) {
	config := input.CephCSIRBD
	params := map[string]string{
		"clusterID":     config.ClusterId,
		"pool":          config.Pool,
		"imageFeatures": config.ImageFeatures,
		GetCSIParamsKey("provisioner-secret-name"):            config.SecretName,      // config.CSIProvisionerSecretName,
		GetCSIParamsKey("provisioner-secret-namespace"):       config.SecretNamespace, // config.CSIProvisionerSecretNamespace,
		GetCSIParamsKey("controller-expand-secret-name"):      config.SecretName,      // config.CSIControllerExpandSecretName,
		GetCSIParamsKey("controller-expand-secret-namespace"): config.SecretNamespace, // config.CSIControllerExpandSecretNamespace,
		GetCSIParamsKey("node-stage-secret-name"):             config.SecretName,      // config.CSINodeStageSecretName,
		GetCSIParamsKey("node-stage-secret-namespace"):        config.SecretNamespace, // config.CSINodeStageSecretNamespace,
		GetCSIParamsKey("fstype"):                             config.CSIFsType,
	}
	return params, nil
}
