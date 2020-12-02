package clusters

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	kubeadmconfig "k8s.io/kubernetes/cmd/kubeadm/app/util/config"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient"
	cloudmod "yunion.io/x/onecloud/pkg/mcclient/modules"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/clusters/addons"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/yunion_host"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	"yunion.io/x/kubecomps/pkg/kubeserver/models/manager"
	"yunion.io/x/kubecomps/pkg/utils/etcd"
	onecloudcli "yunion.io/x/kubecomps/pkg/utils/onecloud/client"
	"yunion.io/x/kubecomps/pkg/utils/registry"
	"yunion.io/x/kubecomps/pkg/utils/ssh"
)

type SYunionHostDriver struct {
	*sClusterAPIDriver
}

func NewYunionHostDriver() models.IClusterDriver {
	return &SYunionHostDriver{
		sClusterAPIDriver: newClusterAPIDriver(api.ModeTypeSelfBuild, api.ProviderTypeOnecloud, api.ClusterResourceTypeHost),
	}
}

func init() {
	models.RegisterClusterDriver(NewYunionHostDriver())
}

func (d *SYunionHostDriver) GetK8sVersions() []string {
	return []string{
		"v1.13.3",
	}
}

func (d *SYunionHostDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.ClusterCreateInput) error {
	if err := d.sClusterAPIDriver.ValidateCreateData(ctx, userCred, ownerId, query, data); err != nil {
		return err
	}
	return yunion_host.ValidateClusterCreateData(data)
}

func GetUsableCloudHosts(s *mcclient.ClientSession) ([]api.UsableInstance, error) {
	params := jsonutils.NewDict()
	filter := jsonutils.NewArray()
	filter.Add(jsonutils.NewString(fmt.Sprintf("host_type.in(%s, %s)", "hypervisor", "kubelet")))
	filter.Add(jsonutils.NewString("host_status.equals(online)"))
	filter.Add(jsonutils.NewString("status.equals(running)"))
	params.Add(filter, "filter")
	result, err := cloudmod.Hosts.List(s, params)
	if err != nil {
		return nil, err
	}
	ret := []api.UsableInstance{}
	for _, host := range result.Data {
		id, _ := host.GetString("id")
		if len(id) == 0 {
			continue
		}
		name, _ := host.GetString("name")
		machine, err := models.MachineManager.GetMachineByResourceId(id)
		if err != nil {
			return nil, err
		}
		if machine != nil {
			continue
		}
		ret = append(ret, api.UsableInstance{
			Id:   id,
			Name: name,
			Type: api.MachineResourceTypeBaremetal,
		})
	}
	return ret, nil
}

func (d *SYunionHostDriver) GetUsableInstances(s *mcclient.ClientSession) ([]api.UsableInstance, error) {
	return GetUsableCloudHosts(s)
}

func (d *SYunionHostDriver) GetKubeconfig(cluster *models.SCluster) (string, error) {
	masterMachine, err := cluster.GetRunningControlplaneMachine()
	if err != nil {
		return "", err
	}
	accessIP, err := masterMachine.GetPrivateIP()
	if err != nil {
		return "", err
	}
	session, err := models.GetAdminSession()
	if err != nil {
		return "", err
	}
	privateKey, err := onecloudcli.GetCloudSSHPrivateKey(session)
	if err != nil {
		return "", err
	}
	out, err := ssh.RemoteSSHCommand(accessIP, 22, "root", "", privateKey, "cat /etc/kubernetes/admin.conf")
	return out, err
}

func (d *SYunionHostDriver) ValidateCreateMachines(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	cluster *models.SCluster,
	repo *api.ImageRepository,
	data []*api.CreateMachineData) error {
	if _, _, err := d.sClusterAPIDriver.ValidateCreateMachines(ctx, userCred, cluster, data); err != nil {
		return err
	}
	return yunion_host.ValidateCreateMachines(data)
}

func (d *SYunionHostDriver) CreateMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *models.SCluster, data []*api.CreateMachineData) ([]manager.IMachine, error) {
	return d.sClusterAPIDriver.CreateMachines(d, ctx, userCred, cluster, data)
}

func (d *SYunionHostDriver) RequestDeployMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *models.SCluster, ms []manager.IMachine, task taskman.ITask) error {
	return d.sClusterAPIDriver.RequestDeployMachines(d, ctx, userCred, cluster, ms, task)
}

func (d *SYunionHostDriver) GetAddonsManifest(cluster *models.SCluster, conf *api.ClusterAddonsManifestConfig) (string, error) {
	commonConf, err := d.GetCommonAddonsConfig(cluster)
	if err != nil {
		return "", err
	}
	reg, err := cluster.GetImageRepository()
	if err != nil {
		return "", err
	}
	pluginConf := &addons.YunionHostPluginsConfig{
		YunionCommonPluginsConfig: commonConf,
		CNIYunionConfig: &addons.CNIYunionConfig{
			YunionAuthConfig: commonConf.CloudProviderYunionConfig.YunionAuthConfig,
			CNIImage:         registry.MirrorImage(reg.Url, "cni", "v2.7.0", ""),
			ClusterCIDR:      cluster.GetServiceCidr(),
		},
	}
	return pluginConf.GenerateYAML()
}

func (d *SYunionHostDriver) GetClusterEtcdEndpoints(cluster *models.SCluster) ([]string, error) {
	ms, err := cluster.GetControlplaneMachines()
	if err != nil {
		return nil, err
	}
	endpoints := []string{}
	for _, m := range ms {
		ip, err := m.GetPrivateIP()
		if err != nil {
			return nil, err
		}
		endpoints = append(endpoints, etcd.GetClientURLByIP(ip))
	}
	return endpoints, nil
}

func (d *SYunionHostDriver) GetClusterEtcdClient(cluster *models.SCluster) (*etcd.Client, error) {
	//spec, err := d.GetClusterAPIClusterSpec(cluster)
	//if err != nil {
	//return nil, err
	//}
	spec, err := cluster.GetEtcdCAKeyPair()
	if err != nil {
		return nil, err
	}
	ca := spec.Certificate
	cert := spec.Certificate
	key := spec.PrivateKey
	if err != nil {
		return nil, err
	}
	endpoints, err := d.GetClusterEtcdEndpoints(cluster)
	if err != nil {
		return nil, err
	}
	return etcd.New(endpoints, ca, cert, key)
}

func (d *SYunionHostDriver) RemoveEtcdMember(etcdCli *etcd.Client, ip string) error {
	// notifies the other members of the etcd cluster about the removing member
	etcdPeerAddress := etcd.GetPeerURL(ip)

	log.Infof("[etcd] get the member id from peer: %s", etcdPeerAddress)
	id, err := etcdCli.GetMemberID(etcdPeerAddress)
	if err != nil {
		return err
	}

	log.Infof("[etcd] removing etcd member: %s, id: %d", etcdPeerAddress, id)
	members, err := etcdCli.RemoveMember(id)
	if err != nil {
		return err
	}
	log.Infof("[etcd] Updated etcd member list: %v", members)
	return nil
}

func (d *SYunionHostDriver) removeKubeadmClusterStatusAPIEndpoint(status *kubeadmapi.ClusterStatus, m manager.IMachine) error {
	ip, err := m.GetPrivateIP()
	if err != nil {
		return err
	}
	for hostname, endpoint := range status.APIEndpoints {
		if hostname == m.GetName() {
			delete(status.APIEndpoints, hostname)
			return nil
		}
		if endpoint.AdvertiseAddress == ip {
			delete(status.APIEndpoints, hostname)
			return nil
		}
	}
	return nil
}

func (d *SYunionHostDriver) updateKubeadmClusterStatus(cli clientset.Interface, status *kubeadmapi.ClusterStatus) error {
	configMap, err := d.getKubeadmConfigmap(cli)
	if err != nil {
		return err
	}
	clusterStatusYaml, err := kubeadmconfig.MarshalKubeadmConfigObject(status)
	if err != nil {
		return err
	}
	configMap.Data[kubeadmconstants.ClusterStatusConfigMapKey] = string(clusterStatusYaml)
	_, err = cli.CoreV1().ConfigMaps(v1.NamespaceSystem).Update(context.Background(), configMap, metav1.UpdateOptions{})
	return err
}

func (d *SYunionHostDriver) RemoveEtcdMembers(cluster *models.SCluster, ms []manager.IMachine) error {
	joinControls := make([]manager.IMachine, 0)
	for _, m := range ms {
		if m.IsControlplane() && !m.IsFirstNode() {
			joinControls = append(joinControls, m)
		}
	}
	if len(joinControls) == 0 {
		return nil
	}
	etcdCli, err := d.GetClusterEtcdClient(cluster)
	if err != nil {
		return err
	}
	defer etcdCli.Cleanup()
	clusterStatus, err := d.GetKubeadmClusterStatus(cluster)
	if err != nil {
		return err
	}
	for _, m := range joinControls {
		ip, err := m.GetPrivateIP()
		if err != nil {
			return err
		}
		if err := d.removeKubeadmClusterStatusAPIEndpoint(clusterStatus, m); err != nil {
			return err
		}
		if err := d.RemoveEtcdMember(etcdCli, ip); err != nil {
			if strings.Contains(err.Error(), "not found") {
				continue
			}
			return err
		}
	}
	cli, err := cluster.GetK8sClient()
	if err != nil {
		return err
	}
	if err := d.updateKubeadmClusterStatus(cli, clusterStatus); err != nil {
		return err
	}
	return nil
}

func (d *SYunionHostDriver) RequestDeleteMachines(ctx context.Context, userCred mcclient.TokenCredential, cluster *models.SCluster, ms []manager.IMachine, task taskman.ITask) error {
	//if err := d.CleanNodeRecords(cluster, ms); err != nil {
	//return err
	//}
	if err := d.RemoveEtcdMembers(cluster, ms); err != nil {
		return err
	}
	return d.sClusterAPIDriver.RequestDeleteMachines(ctx, userCred, cluster, ms, task)
}
