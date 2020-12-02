package addons

import (
	"encoding/base64"

	"yunion.io/x/jsonutils"
)

type CNIYunionConfig struct {
	YunionAuthConfig
	CNIImage    string
	ClusterCIDR string
}

func (c CNIYunionConfig) GenerateYAML() (string, error) {
	return CompileTemplateFromMap(CNIYunionTemplate, c)
}

type CNICalicoConfig struct {
	ControllerImage string
	NodeImage       string
	CNIImage        string
	ClusterCIDR     string
	// EnableNativeIPAlloc will deploy with calico node agent
	EnableNativeIPAlloc bool
	NodeAgentImage      string
}

func (c CNICalicoConfig) GenerateYAML() (string, error) {
	return CompileTemplateFromMap(CNICalicoTemplate, c)
}

type MetricsPluginConfig struct {
	MetricsServerImage string
}

func (c MetricsPluginConfig) GenerateYAML() (string, error) {
	return CompileTemplateFromMap(MetricsTemplate, c)
}

type HelmPluginConfig struct {
	TillerImage string
}

func (c HelmPluginConfig) GenerateYAML() (string, error) {
	return CompileTemplateFromMap(HelmTemplate, c)
}

type YunionAuthConfig struct {
	AuthUrl       string `json:"auth_url"`
	AdminUser     string `json:"admin_user"`
	AdminPassword string `json:"admin_password"`
	AdminProject  string `json:"admin_project"`
	Cluster       string `json:"cluster"`
	InstanceType  string `json:"instance_type"`
	Region        string `json:"region"` // DEP
}

func (c YunionAuthConfig) ToJSONBase64String() string {
	return base64.StdEncoding.EncodeToString([]byte(jsonutils.Marshal(c).PrettyString()))
}

type CloudProviderYunionConfig struct {
	YunionAuthConfig
	CloudProviderImage string
}

func (c CloudProviderYunionConfig) GenerateYAML() (string, error) {
	return CompileTemplateFromMap(YunionCloudProviderTemplate, c)
}

type CSIYunionConfig struct {
	YunionAuthConfig
	AttacherImage    string
	ProvisionerImage string
	PluginImage      string
	RegistrarImage   string
	Base64Config     string
}

func (c CSIYunionConfig) GenerateYAML() (string, error) {
	return CompileTemplateFromMap(CSIYunionTemplate, c)
}

type IngressControllerYunionConfig struct {
	YunionAuthConfig
	Image string
}

func (c IngressControllerYunionConfig) GenerateYAML() (string, error) {
	return CompileTemplateFromMap(YunionIngressControllerTemplate, c)
}
