package api

import (
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
)

const (
	ErrPublicCloudImageNotFound = errors.Error("Image not found")
)

var (
	PublicHypervisors = []string{
		api.HYPERVISOR_ALIYUN,
		// api.HYPERVISOR_AWS,
	}

	OnPremiseHypervisors = []string{
		api.HYPERVISOR_KVM,
		api.HYPERVISOR_BAREMETAL,
		api.HYPERVISOR_ESXI,
	}

	/*
	 * publicCloudImageMap map[string]HypervisorImage = map[string]HypervisorImage{
	 *     // Ref: https://help.aliyun.com/document_detail/100410.html?spm=a2c4g.11186623.6.759.28b92d9frtaFKC
	 *     api.HYPERVISOR_ALIYUN: {
	 *         Id: "centos_7_9_x64_20G_alibase_20201120.vhd",
	 *     },
	 * }
	 */
)

type HypervisorImage struct {
	Id string
}

type MachineCreateConfig struct {
	ImageRepository *ImageRepository       `json:"image_repository"`
	DockerConfig    *DockerConfig          `json:"docker_config"`
	Vm              *MachineCreateVMConfig `json:"vm,omitempty"`
}

type MachineCreateVMConfig struct {
	PreferRegion     string `json:"prefer_region_id"`
	PreferZone       string `json:"prefer_zone_id"`
	PreferWire       string `json:"prefer_wire_id"`
	PreferHost       string `json:"prefer_host_id"`
	PreferBackupHost string `json:"prefer_backup_host"`

	Disks           []*api.DiskConfig           `json:"disks"`
	Networks        []*api.NetworkConfig        `json:"nets"`
	Schedtags       []*api.SchedtagConfig       `json:"schedtags"`
	IsolatedDevices []*api.IsolatedDeviceConfig `json:"isolated_devices"`

	Hypervisor   string `json:"hypervisor"`
	VmemSize     int    `json:"vmem_size"`
	VcpuCount    int    `json:"vcpu_count"`
	InstanceType string `json:"instance_type"`
}

type MachinePrepareInput struct {
	FirstNode bool   `json:"first_node"`
	Role      string `json:"role"`

	// CAKeyPair           *KeyPair `json:"ca_key_pair"`
	// EtcdCAKeyPair       *KeyPair `json:"etcd_ca_key_pair"`
	// FrontProxyCAKeyPair *KeyPair `json:"front_proxy_ca_key_pair"`
	// SAKeyPair           *KeyPair `json:"sa_key_pair"`
	// BootstrapToken string `json:"bootstrap_token"`
	ELBAddress string `json:"elb_address"`

	Config *MachineCreateConfig `json:"config"`

	InstanceId string `json:"-"`
	PrivateIP  string `json:"-"`
}

const (
	DefaultVMMemSize      = 2048       // 2G
	DefaultVMCPUCount     = 2          // 2 core
	DefaultVMRootDiskSize = 100 * 1024 // 100G
)

const (
	MachineMetadataCreateParams = "create_params"
)

type MachineListInput struct {
	apis.VirtualResourceListInput

	// Filter by cluster name or id
	Cluster string `json:"cluster"`
}

type MachineAttachNetworkAddressInput struct {
	// ip_addr specify ip address, e.g. `192.168.0.2`
	IPAddr string `json:"ip_addr"`
}

type CloudMachineInfo struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	Hypervisor string `json:"hypervisor"`
	ZoneId     string `json:"zone_id"`
	NetworkId  string `json:"network_id"`
}

type ClusterMachineCommonInfo struct {
	CloudregionId string
	VpcId         string
}
