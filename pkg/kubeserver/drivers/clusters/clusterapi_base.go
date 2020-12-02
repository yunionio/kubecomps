package clusters

import (
	"context"
	"fmt"

	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmscheme "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/scheme"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	perrors "yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/clusters/addons"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	"yunion.io/x/kubecomps/pkg/kubeserver/models/manager"
	"yunion.io/x/kubecomps/pkg/kubeserver/options"
	"yunion.io/x/kubecomps/pkg/utils/registry"
)

type sClusterAPIDriver struct {
	*SBaseDriver
}

func newClusterAPIDriver(mt api.ModeType, pt api.ProviderType, ct api.ClusterResourceType) *sClusterAPIDriver {
	return &sClusterAPIDriver{
		SBaseDriver: newBaseDriver(mt, pt, ct),
	}
}

func (d *sClusterAPIDriver) NeedGenerateCertificate() bool {
	return true
}

func (d *sClusterAPIDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.ClusterCreateInput) error {
	/*ok, err := clusters.ClusterManager.IsSystemClusterReady()
	if err != nil {
		return err
	}
	if !ok {
		return httperrors.NewNotAcceptableError("System k8s cluster default not running")
	}*/
	return nil
}

func (d *sClusterAPIDriver) CreateMachines(
	drv models.IClusterDriver,
	ctx context.Context,
	userCred mcclient.TokenCredential,
	cluster *models.SCluster,
	data []*api.CreateMachineData,
) ([]manager.IMachine, error) {
	needControlplane, err := cluster.NeedControlplane()
	if err != nil {
		return nil, err
	}
	controls, nodes := drivers.GetControlplaneMachineDatas(cluster.GetId(), data)
	if needControlplane {
		if len(controls) == 0 {
			return nil, fmt.Errorf("Empty controlplane machines")
		}
	}
	cms, nms, err := createMachines(ctx, cluster, userCred, controls, nodes)
	if err != nil {
		return nil, err
	}
	ret := make([]manager.IMachine, 0)
	for _, m := range cms {
		ret = append(ret, m.machine)
	}
	for _, m := range nms {
		ret = append(ret, m.machine)
	}
	return ret, nil
}

type machineData struct {
	machine *models.SMachine
	data    *jsonutils.JSONDict
}

func newMachineData(machine *models.SMachine, input *api.CreateMachineData) *machineData {
	return &machineData{
		machine: machine,
		data:    jsonutils.Marshal(input).(*jsonutils.JSONDict),
	}
}

func createMachines(ctx context.Context, cluster *models.SCluster, userCred mcclient.TokenCredential, controls, nodes []*api.CreateMachineData) ([]*machineData, []*machineData, error) {
	cms := make([]*machineData, 0)
	nms := make([]*machineData, 0)
	cf := func(data []*api.CreateMachineData) ([]*machineData, error) {
		ret := make([]*machineData, 0)
		for _, m := range data {
			obj, err := models.MachineManager.CreateMachineNoHook(ctx, cluster, userCred, m)
			if err != nil {
				return nil, err
			}
			ret = append(ret, newMachineData(obj.(*models.SMachine), m))
		}
		return ret, nil
	}
	var err error
	cms, err = cf(controls)
	if err != nil {
		return nil, nil, err
	}
	nms, err = cf(nodes)
	if err != nil {
		return nil, nil, err
	}
	return cms, nms, nil
}

func machinesPostCreate(ctx context.Context, userCred mcclient.TokenCredential, ms []*machineData) {
	for _, m := range ms {
		func() {
			lockman.LockObject(ctx, m.machine)
			defer lockman.ReleaseObject(ctx, m.machine)
			m.machine.PostCreate(ctx, userCred, userCred, nil, m.data)
		}()
	}
}

type IClusterAPIDriver interface {
	models.IClusterDriver
}

func (d *sClusterAPIDriver) RequestDeployMachines(
	drv models.IClusterDriver,
	ctx context.Context,
	userCred mcclient.TokenCredential,
	cluster *models.SCluster,
	ms []manager.IMachine,
	task taskman.ITask,
) error {
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		var firstCm *models.SMachine
		var restMachines []*models.SMachine
		var needControlplane bool

		doPostCreate := func(m *models.SMachine) {
			lockman.LockObject(ctx, m)
			defer lockman.ReleaseObject(ctx, m)
			m.PostCreate(ctx, userCred, userCred, nil, jsonutils.NewDict())
		}

		for _, m := range ms {
			if m.IsFirstNode() {
				firstCm = m.(*models.SMachine)
				needControlplane = true
			} else {
				restMachines = append(restMachines, m.(*models.SMachine))
			}
		}

		if needControlplane {
			// TODO: fix this
			//masterIP, err := firstCm.GetPrivateIP()
			//if err != nil {
			//log.Errorf("Get privateIP error: %v", err)
			//}
			//if len(masterIP) != 0 {
			//if err := d.updateClusterStaticLBAddress(cluster, masterIP); err != nil {
			//return err
			//}
			//}
			doPostCreate(firstCm)
			// wait first controlplane machine running
			if err := models.WaitMachineRunning(firstCm); err != nil {
				return nil, fmt.Errorf("Create first controlplane machine error: %v", err)
			}
		}

		// create rest join controlplane
		for _, d := range restMachines {
			doPostCreate(d)
		}
		return nil, nil
	})
	return nil
}

func (d *sClusterAPIDriver) GetAddonsManifest(cluster *models.SCluster, conf *api.ClusterAddonsManifestConfig) (string, error) {
	return "", nil
}

func (d *sClusterAPIDriver) ValidateDeleteMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *models.SCluster, ms []manager.IMachine) error {
	oldMachines, err := cluster.GetMachines()
	if err != nil {
		return err
	}
	for _, m := range ms {
		if len(oldMachines) != len(ms) && m.IsFirstNode() {
			return httperrors.NewInputParameterError("First control node %q must deleted at last", m.GetName())
		}
	}
	return nil
}

func (d *sClusterAPIDriver) getKubeadmConfigmap(cli kubernetes.Interface) (*apiv1.ConfigMap, error) {
	configMap, err := cli.CoreV1().ConfigMaps(v1.NamespaceSystem).Get(context.Background(), kubeadmconstants.KubeadmConfigConfigMap, v1.GetOptions{})
	if err != nil {
		return nil, perrors.Wrap(err, "failed to get config map")
	}
	return configMap, nil
}

func (d *sClusterAPIDriver) GetKubeadmClusterStatus(cluster *models.SCluster) (*kubeadmapi.ClusterStatus, error) {
	log.Infof("Reading clusterstatus from cluster: %s", cluster.GetName())
	cli, err := cluster.GetK8sClient()
	if err != nil {
		return nil, err
	}
	configMap, err := d.getKubeadmConfigmap(cli)
	if err != nil {
		return nil, err
	}
	return d.unmarshalClusterStatus(configMap.Data)
}

func (d *sClusterAPIDriver) unmarshalClusterStatus(data map[string]string) (*kubeadmapi.ClusterStatus, error) {
	clusterStatusData, ok := data[kubeadmconstants.ClusterStatusConfigMapKey]
	if !ok {
		return nil, perrors.Errorf("unexpected error when reading kubeadm-config ConfigMap: %s key value pair missing", kubeadmconstants.ClusterStatusConfigMapKey)
	}
	clusterStatus := &kubeadmapi.ClusterStatus{}
	if err := runtime.DecodeInto(kubeadmscheme.Codecs.UniversalDecoder(), []byte(clusterStatusData), clusterStatus); err != nil {
		return nil, err
	}
	return clusterStatus, nil
}

func (d *sClusterAPIDriver) RequestDeleteMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *models.SCluster, ms []manager.IMachine, task taskman.ITask) error {
	items := make([]db.IStandaloneModel, 0)
	for _, m := range ms {
		items = append(items, m.(db.IStandaloneModel))
	}
	return models.MachineManager.StartMachineBatchDeleteTask(ctx, userCred, items, nil, task.GetTaskId())
}

func (d *sClusterAPIDriver) GetAddonYunionAuthConfig(cluster *models.SCluster) (addons.YunionAuthConfig, error) {
	o := options.Options
	s, _ := models.ClusterManager.GetSession()
	authConfig := addons.YunionAuthConfig{
		AuthUrl:       o.AuthURL,
		AdminUser:     o.AdminUser,
		AdminPassword: o.AdminPassword,
		AdminProject:  o.AdminProject,
		Region:        o.Region,
		Cluster:       cluster.GetName(),
		InstanceType:  cluster.ResourceType,
	}
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString("public"), "interface")
	params.Add(jsonutils.NewString(o.Region), "region")
	params.Add(jsonutils.NewString("keystone"), "service")
	ret, err := modules.EndpointsV3.List(s, params)
	if err != nil {
		return authConfig, err
	}
	if len(ret.Data) == 0 {
		return authConfig, perrors.Error("Not found public keystone endpoint")
	}
	authUrl, err := ret.Data[0].GetString("url")
	if err != nil {
		return authConfig, perrors.Wrap(err, "Get public keystone endpoint url")
	}
	authConfig.AuthUrl = authUrl
	return authConfig, nil
}

func (d *sClusterAPIDriver) GetCommonAddonsConfig(cluster *models.SCluster) (*addons.YunionCommonPluginsConfig, error) {
	authConfig, err := d.GetAddonYunionAuthConfig(cluster)
	if err != nil {
		return nil, err
	}
	reg, err := cluster.GetImageRepository()
	if err != nil {
		return nil, err
	}

	commonConf := &addons.YunionCommonPluginsConfig{
		MetricsPluginConfig: &addons.MetricsPluginConfig{
			MetricsServerImage: registry.MirrorImage(reg.Url, "metrics-server-amd64", "v0.3.1", ""),
		},
		/*
		 * HelmPluginConfig: &addons.HelmPluginConfig{
		 *     TillerImage: registry.MirrorImage(reg.Url, "tiller", "v2.11.0", ""),
		 * },
		 */
		CloudProviderYunionConfig: &addons.CloudProviderYunionConfig{
			YunionAuthConfig:   authConfig,
			CloudProviderImage: registry.MirrorImage(reg.Url, "yunion-cloud-controller-manager", "v2.10.0", ""),
		},
		CSIYunionConfig: &addons.CSIYunionConfig{
			YunionAuthConfig: authConfig,
			AttacherImage:    registry.MirrorImage(reg.Url, "csi-attacher", "v1.0.1", ""),
			ProvisionerImage: registry.MirrorImage(reg.Url, "csi-provisioner", "v1.0.1", ""),
			RegistrarImage:   registry.MirrorImage(reg.Url, "csi-node-driver-registrar", "v1.1.0", ""),
			PluginImage:      registry.MirrorImage(reg.Url, "yunion-csi-plugin", "v2.10.0", ""),
			Base64Config:     authConfig.ToJSONBase64String(),
		},
		IngressControllerYunionConfig: &addons.IngressControllerYunionConfig{
			YunionAuthConfig: authConfig,
			Image:            registry.MirrorImage(reg.Url, "yunion-ingress-controller", "v2.10.0", ""),
		},
	}

	return commonConf, nil
}
