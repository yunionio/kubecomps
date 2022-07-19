package api

import "yunion.io/x/onecloud/pkg/apis"

const (
	ClusterComponentCephCSI      = "cephCSI"
	ClusterComponentMonitor      = "monitor"
	ClusterComponentFluentBit    = "fluentbit"
	ClusterComponentMinio        = "minio"
	ClusterComponentMonitorMinio = "monitorMinio"
	ClusterComponentThanos       = "thanos"
)

const (
	ComponentStatusDeploying    = "deploying"
	ComponentStatusDeployFail   = "deploy_fail"
	ComponentStatusDeployed     = "deployed"
	ComponentStatusDeleting     = "deleting"
	ComponentStatusUndeploying  = "undeploying"
	ComponentStatusUndeployFail = "undeploy_fail"
	ComponentStatusDeleteFail   = "delete_fail"
	ComponentStatusUpdating     = "updating"
	ComponentStatusUpdateFail   = "update_fail"
	ComponentStatusInit         = "init"
)

type ComponentCreateInput struct {
	apis.Meta

	Name string `json:"name"`
	Type string `json:"type"`

	Cluster string `json:"cluster"`
	ComponentSettings
}

type ComponentDeleteInput struct {
	apis.Meta

	Name string `json:"name"`
	Type string `json:"type"`
}

type ComponentSettings struct {
	Namespace                 string `json:"namespace"`
	DisableResourceManagement bool   `json:"disableResourceManagement"`
	// Ceph CSI 组件配置
	CephCSI *ComponentSettingCephCSI `json:"cephCSI"`
	// Monitor stack 组件配置
	Monitor *ComponentSettingMonitor `json:"monitor"`
	// Fluentbit 日志收集 agent 配置
	FluentBit *ComponentSettingFluentBit `json:"fluentbit"`
	// Thanos 组件配置
	Thanos *ComponentSettingThanos `json:"thanos"`
	// Minio 对象存储配置
	Minio *ComponentSettingMinio `json:"minio"`
	// Monitor Minio 对象存储配置
	MonitorMinio *ComponentSettingMinio `json:"monitorMinio"`
}

type ComponentCephCSIConfigCluster struct {
	// 集群 Id
	// required: true
	// example: office-ceph-cluster
	ClsuterId string `json:"clusterId"`
	// ceph monitor 连接地址, 比如: 192.168.222.12:6239
	// required: true
	// example: ["192.168.222.12:6239", "192.168.222.13:6239", "192.168.222.14:6239"]
	Monitors []string `json:"monitors"`
}

type ComponentSettingCephCSI struct {
	// 集群配置
	// required: true
	Config []ComponentCephCSIConfigCluster `json:"config"`
}

type ComponentStorage struct {
	// 是否启用持久化存储
	Enabled bool `json:"enabled"`
	// 存储大小, 单位 MB
	SizeMB int `json:"sizeMB"`
	// storageClass 名称
	//
	// required: true
	ClassName string `json:"storageClassName"`
}

func (s ComponentStorage) GetAccessModes() []string {
	return []string{"ReadWriteOnce"}
}

type IngressTLS struct {
	SecretName string `json:"secretName"`
}

type TLSKeyPair struct {
	Name        string `json:"name"`
	Certificate string `json:"certificate"`
	Key         string `json:"key"`
}

type ComponentSettingMonitorGrafanaOAuth struct {
	Enabled           bool   `json:"enabled"`
	ClientId          string `json:"clientId"`
	ClientSecret      string `json:"clientSecret"`
	Scopes            string `json:"scopes"`
	AuthURL           string `json:"authURL"`
	TokenURL          string `json:"tokenURL"`
	APIURL            string `json:"apiURL"`
	AllowedDomains    string `json:"allowedDomains"`
	AllowSignUp       bool   `json:"allowSignUp"`
	RoleAttributePath string `json:"roleAttributePath"`
}

type ComponentSettingMonitorGrafana struct {
	Disable   bool                `json:"disable"`
	Resources *HelmValueResources `json:"resources"`

	// grafana 登录用户名
	// default: admin
	AdminUser string `json:"adminUser"`

	// grafana 登录用户密码
	// default: prom-operator
	AdminPassword string `json:"adminPassword"`
	// grafana 持久化存储配置
	Storage *ComponentStorage `json:"storage"`
	// grafana ingress public address
	PublicAddress string `json:"publicAddress"`
	// grafana ingress host
	Host          string `json:"host"`
	EnforceDomain bool   `json:"enforceDomain"`
	// Ingress expose https key pair
	TLSKeyPair *TLSKeyPair `json:"tlsKeyPair"`
	// Disable subpath /grafana
	DisableSubpath bool   `json:"disableSubpath"`
	Subpath        string `json:"subpath"`
	// Enable thanos query datasource
	EnableThanosQueryDataSource bool                                 `json:"enableThanosQueryDataSource"`
	OAuth                       *ComponentSettingMonitorGrafanaOAuth `json:"oauth"`
	DB                          *DBConfig                            `json:"db"`
}

type DBConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type ComponentSettingMonitorLoki struct {
	Disable   bool                `json:"disable"`
	Resources *HelmValueResources `json:"resources"`

	// loki 持久化存储配置
	Storage *ComponentStorage `json:"storage"`
	// loki s3 对象存储配置
	ObjectStoreConfig *ObjectStoreConfig `json:"objectStoreConfig"`
}

type ComponentSettingMonitorPrometheusThanos struct {
	// thanos sidecar base image, e.g. `thanosio/thanos`
	// BaseImage string `json:"baseImage"`
	// thanos sidecar image version, e.g. `v0.16.0`
	// Version string `json:"version"`
	// thanos sidecar s3 对象存储配置
	ObjectStoreConfig *ObjectStoreConfig `json:"objectStoreConfig"`
}

type ComponentSettingMonitorPrometheus struct {
	Disable   bool                `json:"disable"`
	Resources *HelmValueResources `json:"resources"`
	// prometheus 持久化存储配置
	Storage       *ComponentStorage                        `json:"storage"`
	ThanosSidecar *ComponentSettingMonitorPrometheusThanos `json:"thanosSidecar"`
}

type ComponentSettingVolume struct {
	HostPath  string `json:"hostPath"`
	MountPath string `json:"mountPath"`
}

type ComponentSettingMonitorPromtail struct {
	Disable           bool                   `json:"disable"`
	Resources         *HelmValueResources    `json:"resources"`
	DockerVolumeMount ComponentSettingVolume `json:"dockerVolumeMount"`
	PodsVolumeMount   ComponentSettingVolume `json:"podsVolumeMount"`
}

type HelmValueResource struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

func NewHelmValueResource(cpu string, memory string) *HelmValueResource {
	return &HelmValueResource{
		CPU:    cpu,
		Memory: memory,
	}
}

type HelmValueResources struct {
	Limits   *HelmValueResource `json:"limits"`
	Requests *HelmValueResource `json:"requests"`
}

type ComponentSettingMonitor struct {
	// Grafana 前端日志、监控展示服务
	//
	// required: true
	Grafana *ComponentSettingMonitorGrafana `json:"grafana"`
	// Loki 后端日志收集服务
	//
	// required: true
	Loki *ComponentSettingMonitorLoki `json:"loki"`
	// Prometheus 监控数据采集服务
	//
	// required: true
	Prometheus *ComponentSettingMonitorPrometheus `json:"prometheus"`
	// Promtail 日志收集 agent
	//
	// required: false
	Promtail *ComponentSettingMonitorPromtail `json:"promtail"`
}

type ComponentSettingFluentBitBackendTLS struct {
	// 是否开启 TLS 连接
	//
	// required: false
	TLS bool `json:"tls"`

	// 是否开启 TLS 教研
	//
	// required: false
	TLSVerify bool   `json:"tlsVerify"`
	TLSDebug  bool   `json:"tlsDebug"`
	TLSCA     string `json:"tlsCA"`
}

type ComponentSettingFluentBitBackendForward struct {
	Host string `json:"host"`
	Port int    `json:"port"`
	ComponentSettingFluentBitBackendTLS
}

type ComponentSettingFluentBitBackendCommon struct {
	// 是否启用该后端
	// required: true
	Enabled bool `json:"enabled"`
}

type ComponentSettingFluentBitBackendES struct {
	ComponentSettingFluentBitBackendCommon
	// Elastic 集群连接地址
	//
	// required: true
	// example: 10.168.26.182
	Host string `json:"host"`

	// Elastic 集群连接地址
	//
	// required: true
	// default: 9200
	// example: 9200
	Port int `json:"port"`

	// Elastic index 名称
	//
	// required: true
	// default: fluentbit
	Index string `json:"index"`

	// 类型
	//
	// required: true
	// default: flb_type
	Type string `json:"type"`

	LogstashPrefix string `json:"logstashPrefix"`
	LogstashFormat bool   `json:"logstashFormat"`
	ReplaceDots    bool   `json:"replaceDots"`
	// Optional username credential for Elastic X-Pack access
	HTTPUser string `json:"httpUser"`
	// Password for user defined in HTTPUser
	HTTPPassword string `json:"httpPassword"`
	ComponentSettingFluentBitBackendTLS
}

// check: https://fluentbit.io/documentation/0.14/output/kafka.html
type ComponentSettingFluentBitBackendKafka struct {
	ComponentSettingFluentBitBackendCommon
	// 上报数据格式
	//
	// required: false
	// default: json
	// example: json|msgpack
	Format string `json:"format"`
	// Optional key to store the message
	MessageKey string `json:"messageKey"`
	// Set the key to store the record timestamp
	TimestampKey string `json:"timestampKey"`
	// kafka broker 地址
	//
	// required: true
	// example: ["192.168.222.10:9092", "192.168.222.11:9092", "192.168.222.13:9092"]
	Brokers []string `json:"brokers"`
	// kafka topic
	//
	// required: true
	// example: ["fluent-bit"]
	Topics []string `json:"topics"`
}

const (
	ComponentSettingFluentBitBackendTypeES    = "es"
	ComponentSettingFluentBitBackendTypeKafka = "kafka"
)

type ComponentSettingFluentBitBackend struct {
	// Elasticsearch 配置
	ES *ComponentSettingFluentBitBackendES `json:"es"`
	// Kafka 配置
	Kafka *ComponentSettingFluentBitBackendKafka `json:"kafka"`
}

type ComponentSettingFluentBit struct {
	Backend *ComponentSettingFluentBitBackend `json:"backend"`
}

type ObjectStoreConfig struct {
	// bucket name, e.g. `thanos`
	Bucket string `json:"bucket"`
	// s3 endpoint, e.g. `minio-test.default:9000`
	Endpoint string `json:"endpoint"`
	// access key to auth
	AccessKey string `json:"access_key"`
	// secret key to auth
	SecretKey string `json:"secret_key"`
	// is insecure connection
	Insecure bool `json:"insecure"`
}

type ComponentThanosDnsDiscovery struct {
	// Enabled bool `json:"enabled"`
	// Sidecars service name to discover them using DNS discovery
	// e.g. `prometheus-operated`
	SidecarsService string `json:"sidecarsService"`
	// Sidecars namespace to discover them using DNS discovery
	// e.g. `default`
	SidecarsNamespace string `json:"sidecarsNamespace"`
}

type ComponentThanosQuery struct {
	// LogLevel string `json:"logLevel"`
	// ReplicaLabel []string `json:"replicaLabel"`
	DnsDiscovery ComponentThanosDnsDiscovery `json:"dnsDiscovery"`
	// Statically configure store APIs to connect with Thanos
	Stores []string `json:"stores"`
}

type ComponentThanosCompactor struct {
	Storage ComponentStorage `json:"storage"`
}

type ComponentThanosStoregateway struct {
	Storage ComponentStorage `json:"storage"`
}

type ComponentSettingThanos struct {
	ClusterDomain     string                      `json:"clusterDomain"`
	ObjectStoreConfig ObjectStoreConfig           `json:"objectStoreConfig"`
	Query             ComponentThanosQuery        `json:"query"`
	Store             ComponentThanosStoregateway `json:"storegateway"`
	Compactor         ComponentThanosCompactor    `json:"compactor"`
}

type ComponentMinoMode string

const (
	ComponentMinoModeStandalone  ComponentMinoMode = "standalone"
	ComponentMinoModeDistributed ComponentMinoMode = "distributed"
)

type ComponentSettingMinio struct {
	Mode ComponentMinoMode `json:"mode"`
	// Number of MinIO containers running (aplicable only for MinIO distributed mode)
	Replicas      int `json:"replicas"`
	DrivesPerNode int `json:"drivesPerNode"`
	// Number of zones (aplicable only for MinIO distributed mode)
	Zones int `json:"zones"`
	// Number of drives per node (aplicable only for MinIO distributed mode)
	// Default MinIO admin accessKey
	AccessKey string `json:"accessKey"`
	// Default Minio admin secretKey
	SecretKey string `json:"secretKey"`
	// Default directory mount path, e.g. `/export`
	MountPath string           `json:"mountPath"`
	Storage   ComponentStorage `json:"storage"`
}

type ComponentsStatus struct {
	apis.Meta

	CephCSI      *ComponentStatusCephCSI   `json:"cephCSI"`
	Monitor      *ComponentStatusMonitor   `json:"monitor"`
	FluentBit    *ComponentStatusFluentBit `json:"fluentbit"`
	Thanos       *ComponentStatus          `json:"thanos"`
	Minio        *ComponentStatus          `json:"minio"`
	MonitorMinio *ComponentStatus          `json:"monitorMinio"`
}

type ComponentStatus struct {
	Id      string `json:"id"`
	Created bool   `json:"created"`
	Enabled bool   `json:"enabled"`
	Status  string `json:"status"`
}

type ComponentStatusCephCSI struct {
	ComponentStatus
}

type ComponentStatusMonitor struct {
	ComponentStatus
}

type ComponentStatusFluentBit struct {
	ComponentStatus
}

type ComponentUpdateInput struct {
	apis.Meta

	Type  string `json:"type"`
	Force bool   `json:"force"`

	ComponentSettings
}
