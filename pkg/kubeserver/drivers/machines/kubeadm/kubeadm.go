package kubeadm

import (
	"fmt"

	kubeproxyconfigv1alpha1 "k8s.io/kube-proxy/config/v1alpha1"
	kubeadmv1beta1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"

	"yunion.io/x/log"
)

const (
	// localIPV4lookup looks up the instance's IP through the metadata service.
	// See https://cloudinit.readthedocs.io/en/latest/topics/instancedata.html
	localIPV4Lookup = "{{ ds.meta_data.local_ipv4 }}"

	// HostnameLookup uses the instance metadata service to lookup its own hostname.
	HostnameLookup = "{{ ds.meta_data.hostname }}"

	// ContainerdSocket is the expected path to containerd socket.
	ContainerdSocket = "/var/run/containerd/containerd.sock"

	// APIServerBindPort is the default port for the kube-apiserver to bind to.
	APIServerBindPort = 6443

	// CloudProvider is the name of the cloud provider passed to various
	// kubernetes components.
	CloudProvider = "external"

	nodeRole = "node-role.kubernetes.io/node="
)

type ICluster interface {
	GetAPIServerEndpoint() (string, error)
	GetName() string
	GetServiceDomain() string
	GetServiceCidr() string
	GetVersion() string
	GetPodCidr() string
}

// SetDefaultClusterConfiguration sets default dynamic values without overriding
// user specified values.
func SetDefaultClusterConfiguration(cluster ICluster, base *kubeadmv1beta1.ClusterConfiguration, ipAddress string) (*kubeadmv1beta1.ClusterConfiguration, error) {
	if base == nil {
		base = &kubeadmv1beta1.ClusterConfiguration{}
	}
	out := base.DeepCopy()

	apiServerEndpoint, err := cluster.GetAPIServerEndpoint()
	if err != nil {
		//return nil, errors.Wrapf(err, "Get cluster %s apiServerEndpoint", cluster.GetName())
		// TODO: fix this use LB?
		log.Errorf("Get cluster %s apiServerEndpoint: %v", cluster.GetName(), err)
	}
	if ipAddress == "" {
		ipAddress = localIPV4Lookup
	}
	if apiServerEndpoint == "" {
		apiServerEndpoint = ipAddress
	}
	// Only set the control plane endpoint if the user hasn't specified one
	if out.ControlPlaneEndpoint == "" {
		out.ControlPlaneEndpoint = fmt.Sprintf("%s:%d", apiServerEndpoint, APIServerBindPort)
	}
	// Add the control plane endpoint to the list of cert SAN
	out.APIServer.CertSANs = append(out.APIServer.CertSANs, ipAddress, apiServerEndpoint)
	return out, nil
}

// SetClusterConfigurationOverrides will modify the supplied configuration with certain values
func SetClusterConfigurationOverrides(cluster ICluster, base *kubeadmv1beta1.ClusterConfiguration, ipAddress string) (*kubeadmv1beta1.ClusterConfiguration, error) {
	if base == nil {
		base = &kubeadmv1beta1.ClusterConfiguration{}
	}

	out, err := SetDefaultClusterConfiguration(cluster, base.DeepCopy(), ipAddress)
	if err != nil {
		return nil, err
	}

	if out.APIServer.ControlPlaneComponent.ExtraArgs == nil {
		out.APIServer.ControlPlaneComponent.ExtraArgs = map[string]string{}
	}
	out.APIServer.ControlPlaneComponent.ExtraArgs["cloud-provider"] = CloudProvider

	if out.ControllerManager.ExtraArgs == nil {
		out.ControllerManager.ExtraArgs = map[string]string{}
	}
	out.ControllerManager.ExtraArgs["cloud-provider"] = CloudProvider

	// The kubeadm config clustername must match the provided name of the cluster.
	out.ClusterName = cluster.GetName()
	out.Networking.DNSDomain = cluster.GetServiceDomain()
	out.Networking.PodSubnet = cluster.GetPodCidr()
	out.Networking.ServiceSubnet = cluster.GetServiceCidr()

	// The kubernetes version that kubeadm is using must be the same as the
	// requested versin in the config
	out.KubernetesVersion = cluster.GetVersion()
	return out, nil
}

// SetInitConfigurationOverrides overrides user input on particular fields for
// the kubeadm InitConfiguration.
func SetInitConfigurationOverrides(base *kubeadmv1beta1.InitConfiguration, hostName string) *kubeadmv1beta1.InitConfiguration {
	if base == nil {
		base = &kubeadmv1beta1.InitConfiguration{}
	}
	out := base.DeepCopy()
	if hostName == "" {
		hostName = HostnameLookup
	}
	out.NodeRegistration.Name = hostName
	if out.NodeRegistration.KubeletExtraArgs == nil {
		out.NodeRegistration.KubeletExtraArgs = make(map[string]string)
	}
	out.NodeRegistration.KubeletExtraArgs["cloud-provider"] = CloudProvider
	return out
}

// SetKubeProxyConfigurationOverrides overrides user input on particular fields for
// the kubeadm KubeProxyConfiguration
func SetKubeProxyConfigurationOverrides(base *kubeproxyconfigv1alpha1.KubeProxyConfiguration, clusterCIDR string) *kubeproxyconfigv1alpha1.KubeProxyConfiguration {
	if base == nil {
		base = &kubeproxyconfigv1alpha1.KubeProxyConfiguration{}
	}
	out := base.DeepCopy()
	out.Mode = "ipvs"
	out.IPTables.MasqueradeAll = true
	out.ClusterCIDR = clusterCIDR
	return out
}

// SetJoinNodeConfigurationOverrides overrides user input for certain fields of
// the kubeadm JoinConfiguration during a worker node join.
func SetJoinNodeConfigurationOverrides(
	caCertHash, bootstrapToken, apiServerEndpoint string,
	base *kubeadmv1beta1.JoinConfiguration,
	hostName string,
) *kubeadmv1beta1.JoinConfiguration {
	if base == nil {
		base = &kubeadmv1beta1.JoinConfiguration{}
	}
	out := base.DeepCopy()

	if out.Discovery.BootstrapToken == nil {
		out.Discovery.BootstrapToken = &kubeadmv1beta1.BootstrapTokenDiscovery{}
	}
	out.Discovery.BootstrapToken.APIServerEndpoint = fmt.Sprintf("%s:%d", apiServerEndpoint, APIServerBindPort)
	out.Discovery.BootstrapToken.Token = bootstrapToken
	out.Discovery.BootstrapToken.CACertHashes = append(out.Discovery.BootstrapToken.CACertHashes, caCertHash)

	/*if out.NodeRegistration.Name != "" && out.NodeRegistration.Name != HostnameLookup {

	}*/
	out.NodeRegistration.Name = HostnameLookup
	if hostName != "" {
		out.NodeRegistration.Name = hostName
	}

	if out.NodeRegistration.KubeletExtraArgs == nil {
		out.NodeRegistration.KubeletExtraArgs = map[string]string{}
	}
	out.NodeRegistration.KubeletExtraArgs["cloud-provider"] = CloudProvider
	// out.NodeRegistration.KubeletExtraArgs["node-labels"] = nodeRole
	return out
}

// SetControlPlaneJoinConfigurationOverrides user input for kubeadm join
// configuration during a control plane join action.
func SetControlPlaneJoinConfigurationOverrides(base *kubeadmv1beta1.JoinConfiguration, localIP string) *kubeadmv1beta1.JoinConfiguration {
	if base == nil {
		base = &kubeadmv1beta1.JoinConfiguration{}
	}
	out := base.DeepCopy()

	if out.ControlPlane == nil {
		out.ControlPlane = &kubeadmv1beta1.JoinControlPlane{}
	}
	out.ControlPlane.LocalAPIEndpoint.AdvertiseAddress = localIPV4Lookup
	if localIP != "" {
		out.ControlPlane.LocalAPIEndpoint.AdvertiseAddress = localIP
	}
	out.ControlPlane.LocalAPIEndpoint.BindPort = APIServerBindPort
	return out
}
