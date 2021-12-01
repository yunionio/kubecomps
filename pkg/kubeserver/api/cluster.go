package api

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/apis"
)

// k8s cluster type
type ClusterType string

const (
	// common k8s cluster with nodes
	ClusterTypeDefault ClusterType = "default"
	// nodeless k8s cluster
	// ClusterTypeServerless ClusterType = "serverless"
)

type ModeType string

const (
	// self build k8s cluster
	ModeTypeSelfBuild ModeType = "customize"
	// public cloud managed k8s cluster
	// ModeTypeManaged ModeType = "managed"
	// imported already exists k8s cluster
	ModeTypeImport ModeType = "import"
)

type ProviderType string

const (
	// system provider type means default v3 supervisor cluster
	ProviderTypeSystem ProviderType = "system"
	// default provider type by onecloud
	ProviderTypeOnecloud    ProviderType = "onecloud"
	ProviderTypeOnecloudKvm ProviderType = "kvm"
	// AWS provider
	ProviderTypeAws ProviderType = "aws"
	// Alibaba cloud provider
	ProviderTypeAliyun ProviderType = "aliyun"
	// Azure provider
	ProviderTypeAzure ProviderType = "azure"
	// Tencent cloud provider
	ProviderTypeQcloud ProviderType = "qcloud"
	// External provider type by import
	ProviderTypeExternal ProviderType = "external"
)

const (
	DefaultServiceCIDR   string = "10.43.0.0/16"
	DefaultServiceDomain string = "cluster.local"
	DefaultPodCIDR       string = "10.42.0.0/16"
)

type ClusterResourceType string

const (
	ClusterResourceTypeHost    = "host"
	ClusterResourceTypeGuest   = "guest"
	ClusterResourceTypeUnknown = "unknown"
)

type MachineResourceType string

const (
	MachineResourceTypeBaremetal = "baremetal"
	MachineResourceTypeVm        = "vm"
)

type RoleType string

const (
	RoleTypeControlplane = "controlplane"
	RoleTypeNode         = "node"
)

const (
	MachineStatusInit          = "init"
	MachineStatusCreating      = "creating"
	MachineStatusCreateFail    = "create_fail"
	MachineStatusPrepare       = "prepare"
	MachineStatusPrepareFail   = "prepare_fail"
	MachineStatusRunning       = "running"
	MachineStatusReady         = "ready"
	MachineStatusDeleting      = "deleting"
	MachineStatusDeleteFail    = "delete_fail"
	MachineStatusTerminating   = "terminating"
	MachineStatusTerminateFail = "terminate_fail"

	ClusterStatusInit              = "init"
	ClusterStatusCreating          = "creating"
	ClusterStatusCreateFail        = "create_fail"
	ClusterStatusCreatingMachine   = "creating_machine"
	ClusterStatusCreateMachineFail = "create_machine_fail"
	ClusterStatusDeploying         = "deploying"
	ClusterStatusDeployingFail     = "deploy_fail"
	ClusterStatusRunning           = "running"
	ClusterStatusLost              = "lost"
	ClusterStatusUnknown           = "unknown"
	ClusterStatusError             = "error"
	ClusterStatusDeleting          = "deleting"
	ClusterStatusDeleteFail        = "delete_fail"
)

type ClusterDeployAction string

const (
	ClusterDeployActionCreate     ClusterDeployAction = "create"
	ClusterDeployActionRun        ClusterDeployAction = "run"
	ClusterDeployActionScale      ClusterDeployAction = "scale"
	ClusterDeployActionRemoveNode ClusterDeployAction = "remove-node"
)

type ClusterPreCheckResp struct {
	Pass       bool   `json:"pass"`
	ImageError string `json:"image_error"`
}

type ClusterCreateInput struct {
	apis.StatusDomainLevelResourceCreateInput

	IsSystem               *bool                `json:"is_system"`
	ClusterType            ClusterType          `json:"cluster_type"`
	ResourceType           ClusterResourceType  `json:"resource_type"`
	Mode                   ModeType             `json:"mode"`
	Provider               ProviderType         `json:"provider"`
	ServiceCidr            string               `json:"service_cidr"`
	ServiceDomain          string               `json:"service_domain"`
	PodCidr                string               `json:"pod_cidr"`
	Version                string               `json:"version"`
	HA                     bool                 `json:"ha"`
	Machines               []*CreateMachineData `json:"machines"`
	ImageRepository        *ImageRepository     `json:"image_repository"`
	CloudregionId          string               `json:"cloudregion_id"`
	VpcId                  string               `json:"vpc_id"`
	ManagerId              string               `json:"manager_id"`
	ExternalClusterId      string               `json:"external_cluster_id"`
	ExternalCloudClusterId string               `json:"external_cloud_cluster_id"`

	// imported cluster data
	ImportData *ImportClusterData `json:"import_data"`

	// cluster addons config
	AddonsConfig *ClusterAddonsManifestConfig `json:"addons_config"`
}

type ImageRepository struct {
	// url define cluster image repository url, e.g: registry.hub.docker.com/yunion
	Url string `json:"url"`
	// if insecure, the /etc/docker/daemon.json insecure-registries will add this registry
	Insecure bool `json:"insecure"`
}

type CreateMachineData struct {
	Name         string               `json:"name"`
	ClusterId    string               `json:"cluster_id"`
	Role         string               `json:"role"`
	Provider     string               `json:"provider"`
	ResourceType string               `json:"resource_type"`
	ResourceId   string               `json:"resource_id"`
	Address      string               `json:"address"`
	FirstNode    bool                 `json:"first_node"`
	Config       *MachineCreateConfig `json:"config"`

	ZoneId    string `json:"zone_id"`
	NetworkId string `json:"network_id"`

	// CloudregionId will be inject by cluster
	CloudregionId string `json:"-"`
	VpcId         string `json:"-"`
	// ClusterDeployAction will be inject by background task
	ClusterDeployAction ClusterDeployAction `json:"cluster_deploy_action"`
}

const (
	ImportClusterDistributionK8s       = "k8s"
	ImportClusterDistributionOpenshift = "openshift"
)

type ImportClusterData struct {
	Kubeconfig   string `json:"kubeconfig"`
	ApiServer    string `json:"api_server"`
	Distribution string `json:"distribution"`
	// DistributionInfo should detect by import process
	DistributionInfo ClusterDistributionInfo
}

type ClusterDistributionInfo struct {
	Version string `json:"version"`
}

const (
	ContainerSchedtag = "container"
	DefaultCluster    = "default"
)

type UsableInstance struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type FederatedResourceUsedInput struct {
	// federated resource keyword, e.g: federatednamespace
	FederatedKeyword    string `json:"federated_keyword"`
	FederatedResourceId string `json:"federatedresource_id"`
	// is used by specify federated resource
	FederatedUsed *bool `json:"federated_used"`
}

func (input FederatedResourceUsedInput) ShouldDo() bool {
	return input.FederatedKeyword != "" && input.FederatedResourceId != "" && input.FederatedUsed != nil
}

type ClusterListInput struct {
	apis.StatusDomainLevelResourceListInput
	FederatedResourceUsedInput

	ManagerId     string   `json:"manager_id"`
	Manager       string   `json:"manager" yunion-deprecated-by:"manager_id"`
	CloudregionId string   `json:"cloudregion_id"`
	Provider      []string `json:"provider"`
	Mode          ModeType `json:"mode"`
}

type ClusterSyncInput struct {
	// Force sync
	Force bool `json:"force"`
}

type IClusterRemoteResource interface {
	GetKey() string
}

type ClusterAPIGroupResource struct {
	APIGroup    string             `json:"apiGroup,allowempty"`
	APIResource metav1.APIResource `json:"apiResource"`
}

func (r ClusterAPIGroupResource) GetKey() string {
	return fmt.Sprintf("%s/%s", r.APIGroup, r.APIResource.Name)
}

type IClusterRemoteResources interface {
	GetByIndx(idx int) IClusterRemoteResource
	Len() int
	New() IClusterRemoteResources
	Append(res IClusterRemoteResource) IClusterRemoteResources
	Intersection(ors IClusterRemoteResources) IClusterRemoteResources
	Unionset(ors IClusterRemoteResources) IClusterRemoteResources
}

func operateSet(rs IClusterRemoteResources, ors IClusterRemoteResources, isUnion bool) IClusterRemoteResources {
	marks := make(map[string]IClusterRemoteResource, 0)
	oMarks := make(map[string]IClusterRemoteResource, 0)
	for i := 0; i < rs.Len(); i++ {
		res := rs.GetByIndx(i)
		key := res.GetKey()
		marks[key] = res
	}
	for i := 0; i < ors.Len(); i++ {
		res := ors.GetByIndx(i)
		key := res.GetKey()
		oMarks[key] = res
	}
	for key, res := range oMarks {
		_, ok := marks[key]
		if isUnion {
			if ok {
				continue
			} else {
				marks[key] = res
			}
		} else {
			if ok {
				continue
			} else {
				delete(marks, key)
			}
		}
	}
	ret := rs.New()
	for _, res := range marks {
		ret.Append(res)
	}
	return ret
}

type ClusterAPIGroupResources []ClusterAPIGroupResource

func (rs ClusterAPIGroupResources) New() IClusterRemoteResources {
	ret := make([]ClusterAPIGroupResource, 0)
	return ClusterAPIGroupResources(ret)
}

func (rs ClusterAPIGroupResources) Append(res IClusterRemoteResource) IClusterRemoteResources {
	rs = append(rs, res.(ClusterAPIGroupResource))
	return rs
}

func (rs ClusterAPIGroupResources) GetByIndx(idx int) IClusterRemoteResource {
	return rs[idx]
}

func (rs ClusterAPIGroupResources) Len() int {
	return len(rs)
}

func (rs ClusterAPIGroupResources) operate(ors IClusterRemoteResources, isUnion bool) IClusterRemoteResources {
	return operateSet(rs, ors, isUnion)
}

func (rs ClusterAPIGroupResources) Intersection(ors IClusterRemoteResources) IClusterRemoteResources {
	return rs.operate(ors, false)
}

func (rs ClusterAPIGroupResources) Unionset(ors IClusterRemoteResources) IClusterRemoteResources {
	return rs.operate(ors, true)
}

type ClusterUser struct {
	Name string `json:"name"`
	// FullName is the full name of user
	FullName string `json:"fullName"`
	// Identities are the identities associated with this user
	Identities []string `json:"identities"`
	// Groups specifies group names this user is a member of.
	// This field is deprecated and will be removed in a future release.
	// Instead, create a Group object containing the name of this User.
	Groups []string `json:"groups"`
}

func (user ClusterUser) GetKey() string {
	return fmt.Sprintf("%s/%s", user.Name, user.FullName)
}

type ClusterUsers []ClusterUser

func (rs ClusterUsers) New() IClusterRemoteResources {
	ret := make([]ClusterUser, 0)
	return ClusterUsers(ret)
}

func (rs ClusterUsers) Append(res IClusterRemoteResource) IClusterRemoteResources {
	rs = append(rs, res.(ClusterUser))
	return rs
}

func (rs ClusterUsers) GetByIndx(idx int) IClusterRemoteResource {
	return rs[idx]
}

func (rs ClusterUsers) Len() int {
	return len(rs)
}

func (rs ClusterUsers) operate(ors IClusterRemoteResources, isUnion bool) IClusterRemoteResources {
	return operateSet(rs, ors, isUnion)
}

func (rs ClusterUsers) Intersection(ors IClusterRemoteResources) IClusterRemoteResources {
	return rs.operate(ors, false)
}

func (rs ClusterUsers) Unionset(ors IClusterRemoteResources) IClusterRemoteResources {
	return rs.operate(ors, true)
}

type OptionalNames []string

func (t OptionalNames) String() string {
	return fmt.Sprintf("%v", []string(t))
}

type ClusterUserGroup struct {
	Name string `json:"name"`
	// Users is the list of users in this group.
	Users OptionalNames `json:"users"`
}

func (g ClusterUserGroup) GetKey() string {
	return g.Name
}

type ClusterUserGroups []ClusterUserGroup

func (rs ClusterUserGroups) New() IClusterRemoteResources {
	ret := make([]ClusterUserGroup, 0)
	return ClusterUserGroups(ret)
}

func (rs ClusterUserGroups) Append(res IClusterRemoteResource) IClusterRemoteResources {
	rs = append(rs, res.(ClusterUserGroup))
	return rs
}

func (rs ClusterUserGroups) GetByIndx(idx int) IClusterRemoteResource {
	return rs[idx]
}

func (rs ClusterUserGroups) Len() int {
	return len(rs)
}

func (rs ClusterUserGroups) operate(ors IClusterRemoteResources, isUnion bool) IClusterRemoteResources {
	return operateSet(rs, ors, isUnion)
}

func (rs ClusterUserGroups) Intersection(ors IClusterRemoteResources) IClusterRemoteResources {
	return rs.operate(ors, false)
}

func (rs ClusterUserGroups) Unionset(ors IClusterRemoteResources) IClusterRemoteResources {
	return rs.operate(ors, true)
}

type ClusterPurgeInput struct {
	apis.Meta
	Force bool `json:"force"`
}

type ClusterGetAddonsInput struct {
	EnableNativeIPAlloc bool `json:"enable_native_ip_alloc"`
}

type ClusterAddonNetworkConfig struct {
	EnableNativeIPAlloc bool `json:"enable_native_ip_alloc"`
}

type ClusterAddonsManifestConfig struct {
	Network ClusterAddonNetworkConfig `json:"network"`
}

type ClusterKubesprayConfig struct {
	InventoryContent string               `json:"inventory_content"`
	PrivateKey       string               `json:"private_key"`
	Vars             jsonutils.JSONObject `json:"vars"`
}
