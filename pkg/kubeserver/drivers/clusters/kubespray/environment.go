package kubespray

import (
	"fmt"
	"path"
	"strings"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/constants"
	"yunion.io/x/kubecomps/pkg/kubeserver/options"
)

const (
	DockerVersion_19_03   = "19.03"
	DockerVersion_20_10   = "20.10"
	ContainerdVersion     = "1.4.9"
	NetworkPluginCalico   = "calico"
	NetworkPluginCNI      = "cni"
	CgroupDriver          = "systemd"
	CloudProviderExternal = "external"
)

type IDriver interface {
	ChangeVars(vars *KubesprayVars)
}

func NewDefaultVars(k8sVersion string, extraConf *api.ClusterExtraConfig, drv IDriver) KubesprayVars {
	vars := KubesprayVars{
		DownloadRunOnce: false,
		// YumRepo:                "http://mirrors.aliyun.com",
		// EtcdKubeadmEnabled:     false,
		KubeVersion:            k8sVersion,
		KubeImageRepo:          "registry.aliyuncs.com/google_containers",
		DockerRHRepoBaseUrl:    "https://mirrors.aliyun.com/docker-ce/linux/centos/{{ ansible_distribution_major_version }}/$basearch/stable",
		DockerRHRepoGPGKey:     "https://mirrors.aliyun.com/docker-ce/linux/centos/gpg",
		DockerVersion:          DockerVersion_19_03,
		DockerCliVersion:       DockerVersion_19_03,
		ContainerdVersion:      ContainerdVersion,
		EnableNodelocalDNS:     true,
		NodelocalDNSVersion:    "1.16.0",
		NodelocalDNSImageRepo:  "{{ image_repo }}/k8s-dns-node-cache",
		DNSAutoscalerImageRepo: "{{ image_repo }}/cluster-proportional-autoscaler-{{ image_arch  }}",
		// temporary use kubesphere binary download url check:
		// https://github.com/kubesphere/kubekey/blob/d2a78d20c4a47ab55501ac65f11d54ae51514b1f/pkg/cluster/preinstall/preinstall.go#L50
		KubeletDownloadUrl: "{{ download_file_url }}/kubernetes-release/release/{{ kube_version }}/bin/linux/{{ image_arch }}/kubelet",
		KubectlDownloadUrl: "{{ download_file_url }}/kubernetes-release/release/{{ kube_version }}/bin/linux/{{ image_arch }}/kubectl",
		KubeadmDownloadUrl: "{{ download_file_url }}/kubernetes-release/release/{{ kubeadm_version }}/bin/linux/{{ image_arch }}/kubeadm",
		// CNIBinaryChecksum:  cniChecksum,
		CNIDownloadUrl: "{{ download_file_url }}/plugins/releases/download/{{ cni_version }}/cni-plugins-linux-{{ image_arch }}-{{ cni_version }}.tgz",

		// etcd related vars
		EtcdVersion:                     "v3.4.13",
		EtcdImageRepo:                   "{{ image_repo }}/etcd",
		CalicoctlDownloadUrl:            "{{ download_file_url }}/calicoctl/releases/download/{{ calico_version }}/calicoctl-linux-{{ image_arch }}",
		CalicoCRDsDownloadUrl:           "{{ download_file_url }}/calico/archive/{{ calico_version }}.tar.gz",
		CrictlDownloadUrl:               "{{ download_file_url }}/cri-tools/releases/download/{{ crictl_version }}/crictl-{{ crictl_version }}-{{ ansible_system | lower }}-{{ image_arch }}.tar.gz",
		CalicoNodeImageRepo:             "{{ image_repo }}/calico-node",
		CalicoCNIImageRepo:              "{{ image_repo }}/calico-cni",
		CalicoPolicyImageRepo:           "{{ image_repo }}/calico-kube-controllers",
		CalicoTyphaImageRepo:            "{{ image_repo }}/calico-typha",
		CalicoFlexvolImageRepo:          "{{ image_repo }}/calico-pod2daemon-flexvol",
		CorednsImageIsNamespaced:        false,
		DownloadFileURL:                 options.Options.DownloadFileURL,
		ImageRepo:                       options.Options.ImageRepo,
		DockerUser:                      options.Options.DockerUser,
		DockerPassword:                  options.Options.DockerPassword,
		DockerHost:                      options.Options.DockerHost,
		AutoRenewCertificates:           true,
		NginxImageRepo:                  "{{ image_repo }}/nginx",
		NginxImageTag:                   "1.19",
		IngressNginxEnabled:             true,
		IngressNginxControllerImageRepo: "{{ kube_image_repo }}/nginx-ingress-controller",
		KubeNetworkPlugin:               NetworkPluginCalico,
		KubeletCgroupDriver:             CgroupDriver,
		DockerCgroupDriver:              CgroupDriver,
		OverrideSystemHostname:          true,
	}

	if extraConf != nil {
		vars.DockerInsecureRegistries = extraConf.DockerInsecureRegistries
		vars.DockerRegistryMirrors = extraConf.DockerRegistryMirrors
	}
	vars.DockerOptions = "--bridge=none --live-restore"

	if strings.Compare(k8sVersion, "v1.21.0") >= 0 {
		vars.CNIVersion = constants.CNI_VERSION_1_22_9
		vars.CalicoVersion = constants.CALICO_VERSION_1_22_9
		vars.KubesprayVersion = constants.KUBESPRAY_VERSION_1_22_9
		vars.ContainerManager = "docker"
		vars.ContainerdVersion = constants.CONTAINERD_VERSION_1_22_9
		vars.IngressNginxControllerImageTag = constants.NGINX_INGRESS_CONTROLLER_1_22_9
		vars.EtcdVersion = constants.ETCD_VERSION_1_22_9
		calicoctlDownloadUrl := "{{ download_file_url }}/calico/releases/download/{{ calico_version }}/calicoctl-linux-{{ image_arch }}"
		vars.CalicoctlDownloadUrl = calicoctlDownloadUrl
		vars.CalicoctlAlternateDownloadUrl = calicoctlDownloadUrl
		vars.EtcdDownloadUrl = "{{ download_file_url }}/etcd/releases/download/{{ etcd_version }}/etcd-{{ etcd_version }}-linux-{{ image_arch }}.tar.gz"
	} else if strings.Compare(k8sVersion, "v1.19.0") >= 0 {
		vars.CNIVersion = constants.CNI_VERSION_1_20_0
		vars.CalicoVersion = constants.CALICO_VERSION_1_20_0
		vars.KubesprayVersion = constants.KUBESPRAY_VERSION_1_20_0
		vars.IngressNginxControllerImageTag = constants.NGINX_INGRESS_CONTROLLER_1_20_0
	} else {
		vars.CNIVersion = constants.CNI_VERSION_1_17_0
		vars.CalicoVersion = constants.CALICO_VERSION_1_17_0
		vars.KubesprayVersion = constants.KUBESPRAY_VERSION_1_17_0
		vars.IngressNginxControllerImageTag = constants.NGINX_INGRESS_CONTROLLER_1_17_0
	}
	vars.IngressNginxConfigmap = map[string]string{
		// https://kubernetes.github.io/ingress-nginx/user-guide/tls/#legacy-tls
		"ssl-ciphers":   "ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384:DHE-RSA-CHACHA20-POLY1305:ECDHE-ECDSA-AES128-SHA256:ECDHE-RSA-AES128-SHA256:ECDHE-ECDSA-AES128-SHA:ECDHE-RSA-AES128-SHA:ECDHE-ECDSA-AES256-SHA384:ECDHE-RSA-AES256-SHA384:ECDHE-ECDSA-AES256-SHA:ECDHE-RSA-AES256-SHA:DHE-RSA-AES128-SHA256:DHE-RSA-AES256-SHA256:AES128-GCM-SHA256:AES256-GCM-SHA384:AES128-SHA256:AES256-SHA256:AES128-SHA:AES256-SHA:DES-CBC3-SHA",
		"ssl-protocols": "TLSv1.2 TLSv1.3",
	}
	drv.ChangeVars(&vars)
	return vars
}

func NewOfflineVars(k8sVersion string, extraConf *api.ClusterExtraConfig, drv IDriver) KubesprayVars {
	vars := NewDefaultVars(k8sVersion, extraConf, drv)
	globalOpt := options.Options

	registryUrl := globalOpt.OfflineRegistryServiceURL
	if registryUrl != "" {
		// image repo configuration
		if vars.DockerInsecureRegistries == nil {
			vars.DockerInsecureRegistries = make([]string, 0)
		}
		vars.DockerInsecureRegistries = append(vars.DockerInsecureRegistries, registryUrl)
		yunionRepo := path.Join(registryUrl, "yunionio")
		vars.KubeImageRepo = yunionRepo
		vars.ImageRepo = yunionRepo
	}

	nginxUrl := globalOpt.OfflineNginxServiceURL
	if nginxUrl != "" {
		filesUrl := nginxUrl + "/files"

		// kubernetes-release configuration
		k8sFileUrl := filesUrl + "/storage.googleapis.com"
		vars.KubeletDownloadUrl = k8sFileUrl + "/kubernetes-release/release/{{ kube_version }}/bin/linux/{{ image_arch }}/kubelet"
		vars.KubectlDownloadUrl = k8sFileUrl + "/kubernetes-release/release/{{ kube_version }}/bin/linux/{{ image_arch }}/kubectl"
		vars.KubeadmDownloadUrl = k8sFileUrl + "/kubernetes-release/release/{{ kubeadm_version }}/bin/linux/{{ image_arch }}/kubeadm"

		// calico configuration
		calicoFileUrl := filesUrl + "/github.com/projectcalico"
		vars.CalicoctlDownloadUrl = calicoFileUrl + "/calicoctl/releases/download/{{ calico_version }}/calicoctl-linux-{{ image_arch }}"
		vars.CalicoCRDsDownloadUrl = calicoFileUrl + "/calico/archive/{{ calico_version }}.tar.gz"

		// cri-tools
		vars.CrictlDownloadUrl = filesUrl + "/github.com/kubernetes-sigs/cri-tools/releases/download/{{ crictl_version }}/crictl-{{ crictl_version }}-{{ ansible_system | lower }}-{{ image_arch }}.tar.gz"

		// docker-ce package configuration
		vars.DockerRHRepoBaseUrl = fmt.Sprintf("%s/rpms/", nginxUrl)
		vars.DockerRHRepoGPGKey = ""

		// cni
		vars.CNIDownloadUrl = filesUrl + "/github.com/containernetworking/plugins/releases/download/{{ cni_version }}/cni-plugins-linux-{{ image_arch }}-{{ cni_version }}.tgz"
	}

	return vars
}
