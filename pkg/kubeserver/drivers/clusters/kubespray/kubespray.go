package kubespray

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"golang.org/x/crypto/ssh"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"
)

type KubesprayNodeRole string

const (
	KubesprayNodeRoleMaster   = "kube-master"
	KubesprayNodeRoleEtcd     = "etcd"
	KubesprayNodeRoleNode     = "kube-node"
	KubesprayNodeRoleCalicoRR = "calico-rr"
)

var (
	DefaultKubesprayClusterYML        = "cluster.yml"
	DefaultKubesprayRemoveNodeYML     = "remove-node.yml"
	DefaultKubesprayScaleYML          = "scale.yml"
	DefaultKubesprayUpgradeClusterYML = "upgrade-cluster.yml"

	KubesprayNodeRoles = []KubesprayNodeRole{
		KubesprayNodeRoleMaster,
		KubesprayNodeRoleEtcd,
		KubesprayNodeRoleNode,
		KubesprayNodeRoleCalicoRR,
	}
)

type KubesprayVars struct {
	// DownloadRunOnce will make kubespray download container images and binaries only once
	// and then push them to the cluster nodes. The default download delegate node is the
	// first `kube-master`
	DownloadRunOnce bool `json:"download_run_once"`
	// YumRepo for rpm: http://mirrors.aliyun.com
	YumRepo string `json:"yum_repo"`
	// GCRImageRepo: gcr.azk8s.cn
	GCRImageRepo string `json:"gcr_image_repo"`
	// KubeImageRepo: registry.aliyuncs.com/google_containers or gcr.azk8s.cn/google-containers
	KubeImageRepo string `json:"kube_image_repo"`
	// QuayImageRepo: quay.mirrors.ustc.edu.cn
	QuayImageRepo string `json:"quay_image_repo"`

	// Docker CentOS/Redhat repo
	// DockerRHRepoBaseUrl: {{ yum_repo }}/docker-ce/$releasever/$basearch
	DockerRHRepoBaseUrl string `json:"docker_rh_repo_base_url"`
	// DockerRHRepoGPGKey: {{ yum_repo }}/docker-ce/gpg
	DockerRHRepoGPGKey string `json:"docker_rh_repo_gpgkey"`

	// kubespray etcd cluster not support kubeadm managed very well currently
	// EtcdKubeadmEnabled     bool   `json:"etcd_kubeadm_enabled"`
	KubeVersion            string `json:"kube_version"`
	CNIVersion             string `json:"cni_version"`
	EnableNodelocalDNS     bool   `json:"enable_nodelocaldns"`
	NodelocalDNSVersion    string `json:"nodelocaldns_version"`
	NodelocalDNSImageRepo  string `json:"nodelocaldns_image_repo"`
	DNSAutoscalerImageRepo string `json:"dnsautoscaler_image_repo"`
	// KubeletDownloadUrl: https://storage.googleapis.com/kubernetes-release/release/{{ kube_version  }}/bin/linux/{{ image_arch }}/kubelet
	KubeletDownloadUrl string `json:"kubelet_download_url"`
	// KubectlDownloadUrl: https://storage.googleapis.com/kubernetes-release/release/{{ kube_version }}/bin/linux/{{ image_arch }}/kubectl
	KubectlDownloadUrl string `json:"kubectl_download_url"`
	// KubeadmDownloadUrl: https://storage.googleapis.com/kubernetes-release/release/{{ kubeadm_version }}/bin/linux/{{ image_arch }}/kubeadm
	KubeadmDownloadUrl string `json:"kubeadm_download_url"`
	// CNIDownloadUrl: https://github.com/containernetworking/plugins/releases/download/{{ cni_version }}/cni-plugins-linux-{{ image_arch }}-{{ cni_version }}.tgz
	CNIDownloadUrl    string `json:"cni_download_url"`
	CNIBinaryChecksum string `json:"cni_binary_checksum"`
	// CrictlDownloadUrl: https://iso.yunion.cn/binaries/cri-tools/releases/download/{{ crictl_version }}/crictl-{{ crictl_version }}-{{ ansible_system | lower }}-{{ image_arch }}.tar.gz
	CrictlDownloadUrl string `json:"crictl_download_url"`

	// etcd related vars
	EtcdImageRepo string `json:"etcd_image_repo"`
	EtcdVersion   string `json:"etcd_version"`

	// Calico related vars
	// CalicoctlDownloadUrl: https://iso.yunion.cn/binaries/calicoctl/releases/download/v3.16.5/calicoctl-linux-amd64
	CalicoVersion         string `json:"calico_version"`
	CalicoctlDownloadUrl  string `json:"calicoctl_download_url"`
	CalicoNodeImageRepo   string `json:"calico_node_image_repo"`
	CalicoNodeImageTag    string `json:"calico_node_image_tag"`
	CalicoCNIImageRepo    string `json:"calico_cni_image_repo"`
	CalicoCNIImageTag     string `json:"calico_cni_image_tag"`
	CalicoPolicyImageRepo string `json:"calico_policy_image_repo"`
	CalicoPolicyImageTag  string `json:"calico_policy_image_tag"`
	CalicoTyphaImageRepo  string `json:"calico_typha_image_repo"`
	CalicoTyphaImageTag   string `json:"calico_typha_image_tag"`

	// Address in cert sans
	SupplementaryAddresses []string `json:"supplementary_addresses_in_ssl_keys"`

	// Add new master and etcd node vars
	IgnoreAssertErrors string `json:"ignore_assert_errors"`
	EtcdRetries        int    `json:"etcd_retries"`

	// Remove nodes vars
	// ResetNodes should be false if want to remove node not online
	ResetNodes bool `json:"reset_nodes"`
	// Node being removed
	Node string `json:"node"`
	// DeleteNodesConfirmation must set to yes
	DeleteNodesConfirmation string `json:"delete_nodes_confirmation"`
	//kubespray verion
	KubesprayVersion string `json:"kubespray_version"`
	//CorednsImage path /coredns
	CorednsImageIsNamespaced bool   `json:"coredns_image_is_namespaced"`
	DownloadFileAddr         string `json:"download_file_addr"`
}

func (v KubesprayVars) Validate() error {
	/*
	 * if err := ValidateKubernetesVersion(v.KubeVersion); err != nil {
	 *     return ErrKubernetesVersionEmpty
	 * }
	 */

	return nil
}

type KubesprayInventoryHost struct {
	Hostname       string
	AnsibleHost    string
	User           string
	Ip             string
	AccessIp       string
	Password       string
	Roles          sets.String
	privateKey     string
	privateKeyFile string
}

func NewKubesprayInventoryHost(
	hostname string,
	sshIP string,
	user string,
	password string,
	// accessIP string,
	// privateIp string,
	roles ...KubesprayNodeRole,
) (*KubesprayInventoryHost, error) {

	if len(roles) == 0 {
		return nil, errors.Error("role not provide")
	}

	rs := sets.NewString()
	for _, r := range roles {
		rs.Insert(string(r))
	}

	return &KubesprayInventoryHost{
		Hostname:    hostname,
		AnsibleHost: sshIP,
		User:        user,
		Password:    password,
		Roles:       rs,
		// AccessIp:    accessIP,
		// Ip:          privateIp,
	}, nil
}

func (h KubesprayInventoryHost) IsEtcdMember() bool {
	return h.HasRole(KubesprayNodeRoleEtcd)
}

func (h KubesprayInventoryHost) GetEtcdMemberName() string {
	if !h.IsEtcdMember() {
		return ""
	}
	return h.Hostname
}

func (h KubesprayInventoryHost) HasRole(role KubesprayNodeRole) bool {
	return h.Roles.Has(string(role))
}

func (h *KubesprayInventoryHost) GetPrivateKey() string {
	return h.privateKey
}

func (h *KubesprayInventoryHost) SetPrivateKey(content []byte) error {
	if _, err := ssh.ParsePrivateKey(content); err != nil {
		return errors.Wrap(err, "invalid ssh privateKey")
	}

	tf, err := ioutil.TempFile(os.TempDir(), h.AnsibleHost)
	if err != nil {
		return errors.Wrap(err, "new temporary file")
	}
	defer tf.Close()

	if _, err := tf.Write(content); err != nil {
		return errors.Wrap(err, "write content to file")
	}

	fPath := tf.Name()
	h.privateKey = string(content)
	h.privateKeyFile = fPath
	// clear password
	h.Password = ""
	return nil
}

func (h *KubesprayInventoryHost) Clear() error {
	if h.privateKeyFile != "" {
		if err := os.Remove(h.privateKeyFile); err != nil {
			if !os.IsNotExist(err) {
				return errors.Wrapf(err, "remove privateKeyFile %s", h.privateKeyFile)
			}
		}
		h.privateKey = ""
		h.privateKeyFile = ""
	}

	return nil
}

func (h KubesprayInventoryHost) ToString() (string, error) {
	if h.Hostname == "" {
		return "", errors.Error("hostname is empty")
	}

	if h.AnsibleHost == "" {
		return "", errors.Error("host is empty")
	}
	/*
	 * if h.Password == "" && h.privateKeyFile == "" {
	 *     return "", errors.Error("password or privateKey is empty")
	 * }
	 */
	if h.User == "" {
		return "", errors.Error("user is empty")
	}
	out := fmt.Sprintf("%s", h.Hostname)
	out = fmt.Sprintf("%s\tansible_host=%s", out, h.AnsibleHost)
	out = fmt.Sprintf("%s\tansible_ssh_user=%s", out, h.User)
	if h.Password != "" {
		out = fmt.Sprintf("%s\tansible_ssh_pass=%s", out, h.Password)
	}
	if h.privateKeyFile != "" {
		out = fmt.Sprintf("%s\tansible_ssh_private_key_file=%s", out, h.privateKeyFile)
	}

	if h.Ip != "" {
		out = fmt.Sprintf("%s\tip=%s", out, h.Ip)
	}
	if h.AccessIp != "" {
		out = fmt.Sprintf("%s\taccess_ip=%s", out, h.AccessIp)
	}

	etcdMemberName := h.GetEtcdMemberName()
	if etcdMemberName != "" {
		out = fmt.Sprintf("%s\tetcd_member_name=%s", out, etcdMemberName)
	}

	return out, nil
}

type KubesprayInventory struct {
	Hosts []*KubesprayInventoryHost
}

func NewKubesprayInventory(host ...*KubesprayInventoryHost) *KubesprayInventory {
	return &KubesprayInventory{
		Hosts: host,
	}
}

func (i KubesprayInventory) IsIncludeHost(host string) bool {
	for _, h := range i.Hosts {
		if h.Hostname == host {
			return true
		}
	}
	return false
}

func (i KubesprayInventory) ToString() (string, error) {
	if len(i.Hosts) == 0 {
		return "", errors.Error("hosts is empty")
	}

	out := new(bytes.Buffer)

	roleGroups := map[KubesprayNodeRole][]string{}

	io.WriteString(out, "[all]\n")

	for idx := range i.Hosts {
		h := i.Hosts[idx]

		if h.Roles.Len() == 0 {
			return "", errors.Errorf("host %s no roles", h.Hostname)
		}

		hStr, err := h.ToString()
		if err != nil {
			return "", errors.Wrapf(err, "host %s", h.Hostname)
		}

		io.WriteString(out, hStr+"\n")

		for _, checkRole := range KubesprayNodeRoles {
			if h.HasRole(checkRole) {
				log.Infof("h %s has role %s", h.Hostname, checkRole)
				grp := roleGroups[checkRole]
				grp = append(grp, h.Hostname)
				roleGroups[checkRole] = grp
			}
		}
	}
	io.WriteString(out, "\n")

	etcdNodes := roleGroups[KubesprayNodeRoleEtcd]
	if len(etcdNodes) == 0 {
		return "", errors.Error("etcd nodes is empty")
	}

	for _, checkRole := range KubesprayNodeRoles {
		io.WriteString(out, fmt.Sprintf("[%s]\n", checkRole))
		for _, name := range roleGroups[checkRole] {
			io.WriteString(out, name+"\n")
		}
		io.WriteString(out, "\n")
	}

	io.WriteString(out, "[k8s-cluster:children]\n")
	io.WriteString(out, KubesprayNodeRoleMaster+"\n")
	io.WriteString(out, KubesprayNodeRoleNode+"\n")
	io.WriteString(out, KubesprayNodeRoleCalicoRR)

	return out.String(), nil
}
