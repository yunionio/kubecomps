package components

import (
	"fmt"

	"k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
)

type Image struct {
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
}

type Image2 struct {
	// e.g. `docker.io`
	Registry string `json:"registry"`
	// e.g. `bitnami/thanos`
	Repository string `json:"repository"`
	// e.g. `v0.16.0`
	Tag string `json:"tag"`
}

type PromTLSProxy struct {
	Enabled bool  `json:"enabled"`
	Image   Image `json:"image"`
}

type PromAdmissionWebhooksPatch struct {
	Enabled bool `json:"enabled"`
}

type PromAdmissionWebhooks struct {
	Enabled bool                       `json:"enabled"`
	Patch   PromAdmissionWebhooksPatch `json:"patch"`
}

type Prometheus struct {
	Enabled bool           `json:"enabled"`
	Spec    PrometheusSpec `json:"prometheusSpec"`
}

type Resources struct {
	Requests map[string]string `json:"requests"`
}

type PersistentVolumeClaimSpec struct {
	StorageClassName *string   `json:"storageClassName"`
	AccessModes      []string  `json:"accessModes"`
	Resources        Resources `json:"resources"`
}

type PersistentVolumeClaim struct {
	Spec PersistentVolumeClaimSpec `json:"spec"`
}

type PrometheusStorageSpec struct {
	Template PersistentVolumeClaim `json:"volumeClaimTemplate"`
}

func NewPrometheusStorageSpec(storage api.ComponentStorage) (*PrometheusStorageSpec, error) {
	sizeGB := storage.SizeMB / 1024
	if sizeGB <= 0 {
		return nil, httperrors.NewInputParameterError("size must large than 1GB")
	}
	storageSize := fmt.Sprintf("%dGi", sizeGB)
	spec := new(PersistentVolumeClaimSpec)
	if storage.ClassName != "" {
		spec.StorageClassName = &storage.ClassName
	}
	spec.AccessModes = storage.GetAccessModes()
	spec.Resources = Resources{
		Requests: map[string]string{
			"storage": storageSize,
		},
	}
	return &PrometheusStorageSpec{
		Template: PersistentVolumeClaim{
			Spec: *spec,
		},
	}, nil
}

type ObjectStorageConfig struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type ThanosSidecarSpec struct {
	BaseImage           string              `json:"baseImage"`
	Version             string              `json:"version"`
	ObjectStorageConfig ObjectStorageConfig `json:"objectStorageConfig"`
}

type PrometheusSpec struct {
	CommonConfig
	// image: quay.io/prometheus/prometheus:v2.15.2
	Image       Image                  `json:"image"`
	StorageSpec *PrometheusStorageSpec `json:"storageSpec"`
	// How long to retain metrics
	Retention string `json:"retention"`
	// Maximum size of metrics
	RetentionSize string `json:"retentionSize"`
	// Resource limits & requests
	Resources Resources `json:"resources"`
	// ThanosSpec defines parameters for a Prometheus server within a Thanos sidecar.
	Thanos ThanosSidecarSpec `json:"thanos"`
}

type Alertmanager struct {
	Enabled bool             `json:"enabled"`
	Spec    AlertmanagerSpec `json:"alertmanagerSpec"`
}

type AlertmanagerSpec struct {
	CommonConfig
	// image: quay.io/prometheus/alertmanager:v0.20.0
	Image Image `json:"image"`
}

type NodeExporter struct {
	Enabled bool `json:"enabled"`
}

type PrometheusNodeExporter struct {
	Enabled bool `json:"enabled"`
	// image: quay.io/prometheus/node-exporter:v0.18.1
	Image Image `json:"image"`
}

type KubeStateMetrics struct {
	CommonConfig
	// image: quay.io/coreos/kube-state-metrics:v1.9.4
	Image Image `json:"image"`
}

type GrafanaSidecarDataSources struct {
	DefaultDatasourceEnabled bool `json:"defaultDatasourceEnabled"`
}

type GrafanaSidecarDashboards struct {
	Enabled bool `json:"enabled"`
}

type GrafanaSidecar struct {
	// image: kiwigrid/k8s-sidecar:0.1.99
	Image       Image                     `json:"image"`
	Dashboards  GrafanaSidecarDashboards  `json:"dashboards"`
	Datasources GrafanaSidecarDataSources `json:"datasources"`
}

type Storage struct {
	Type             string   `json:"type"`
	Enabled          bool     `json:"enabled"`
	StorageClassName string   `json:"storageClassName"`
	StorageClass     string   `json:"storageClass"`
	AccessModes      []string `json:"accessModes"`
	Size             string   `json:"size"`
}

func NewPVCStorage(storage *api.ComponentStorage) (*Storage, error) {
	sizeMB := storage.SizeMB
	sizeGB := sizeMB / 1024
	if sizeGB <= 0 {
		return nil, httperrors.NewInputParameterError("size must large than 1GB")
	}
	accessModes := storage.GetAccessModes()
	return &Storage{
		Type:             "pvc",
		Enabled:          true,
		StorageClassName: storage.ClassName,
		StorageClass:     storage.ClassName,
		AccessModes:      accessModes,
		Size:             fmt.Sprintf("%dGi", sizeGB),
	}, nil
}

type Service struct {
	Type     string `json:"type"`
	NodePort string `json:"nodePort"`
}

type GrafanaIngress struct {
	Enabled bool            `json:"enabled"`
	Path    string          `json:"path"`
	Host    string          `json:"host,allowempty"`
	Secret  *api.TLSKeyPair `json:"secret"`

	TLS []*api.IngressTLS `json:"tls"`
}

type GrafanaIniServer struct {
	RootUrl          string `json:"root_url"`
	ServeFromSubPath bool   `json:"serve_from_sub_path"`
	Domain           string `json:"domain,omitempty"`
	EnforceDomain    bool   `json:"enforce_domain,omitempty"`
	HttpPort         string `json:"http_port"`
	Protocol         string `json:"protocol"`
}

type GrafanaIniOAuth struct {
	Enabled                 bool   `json:"enabled"`
	ClientId                string `json:"client_id"`
	ClientSecret            string `json:"client_secret"`
	Scopes                  string `json:"scopes"`
	AuthURL                 string `json:"auth_url"`
	TokenURL                string `json:"token_url"`
	APIURL                  string `json:"api_url"`
	AllowedDomains          string `json:"allowed_domains"`
	AllowSignUp             bool   `json:"allow_sign_up"`
	RoleAttributePath       string `json:"role_attribute_path"`
	IdTokenAttributeName    string `json:"id_token_attribute_name"`
	TlsSkipVerifyInsecure   bool   `json:"tls_skip_verify_insecure"`
	AllowAssignGrafanaAdmin bool   `json:"allow_assign_grafana_admin"`
}

type GrafanaIniDatabase struct {
	Type     string `json:"type"`
	Host     string `json:"host"`
	Name     string `json:"name"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type GrafanaIniUsers struct {
	DefaultTheme string `json:"default_theme"`
}

type GrafanaIniSecurity struct {
	CookieSamesite string `json:"cookie_samesite"`
	CookieSecure   bool   `json:"cookie_secure"`
	AllowEmbedding bool   `json:"allow_embedding"`
}

type GrafanaIniAuth struct {
	LoginCookieName string `json:"login_cookie_name"`
}

type GrafanaIni struct {
	Server   *GrafanaIniServer   `json:"server"`
	OAuth    *GrafanaIniOAuth    `json:"auth.generic_oauth"`
	Database *GrafanaIniDatabase `json:"database"`
	Users    *GrafanaIniUsers    `json:"users"`
	Security *GrafanaIniSecurity `json:"security"`
	Auth     *GrafanaIniAuth     `json:"auth"`
}

type GrafanaDataSourceJsonData struct {
	// TlsAuth       bool `json:"tlsAuth"`
	TlsSkipVerify bool `json:"tlsSkipVerify"`
}

type GrafanaAdditionalDataSource struct {
	Name      string                     `json:"name"`
	Type      string                     `json:"type"`
	Url       string                     `json:"url"`
	Access    string                     `json:"access"`
	IsDefault bool                       `json:"isDefault"`
	Database  string                     `json:"database,omitempty"`
	JsonData  *GrafanaDataSourceJsonData `json:"jsonData,omitempty"`
}

type GrafanaAdditionalDataSources []GrafanaAdditionalDataSource

type Grafana struct {
	CommonConfig
	AdminUser     string         `json:"adminUser"`
	AdminPassword string         `json:"adminPassword"`
	Sidecar       GrafanaSidecar `json:"sidecar"`
	// image: grafana/grafana:6.7.1
	Image                 Image                        `json:"image"`
	Storage               *Storage                     `json:"persistence"`
	Service               *Service                     `json:"service"`
	Ingress               *GrafanaIngress              `json:"ingress"`
	GrafanaIni            *GrafanaIni                  `json:"grafana.ini"`
	AdditionalDataSources GrafanaAdditionalDataSources `json:"additionalDataSources"`
}

type LokiSchemaConfigIndex struct {
	Prefix string `json:"prefix"`
	Period string `json:"period"`
}

type LokiSchemaConfig struct {
	From        string                `json:"from"`
	Store       string                `json:"store"`
	ObjectStore string                `json:"object_store"`
	Schema      string                `json:"schema"`
	Index       LokiSchemaConfigIndex `json:"index"`
}

type LokiConfigSchemaConfig struct {
	Configs []LokiSchemaConfig `json:"configs"`
}

type LokiStorageConfigBoltdbShipper struct {
	// e.g.: `/data/loki/boltdb-shipper-active`
	// ActiveIndexDirectory string `json:"active_index_directory"`
	// e.g.: `/data/loki/boltdb-shipper-cache`
	// CacheLocation string `json:"cache_location"`
	// e.g. `24h`
	CacheTTL string `json:"cache_ttl"`
	// e.g. `s3`
	SharedStore string `json:"shared_store"`
}

type LokiStorageConfigAws struct {
	// e.g. `s3://plUbSwTzWXi3QsP0B8Ab:Rp40yaVS7NVf4zkrpIU6WANlbxWQTUErSIs1EduG@minio-test.default:9000/loki-bucket`
	S3               string `json:"s3"`
	S3ForcepathStyle bool   `json:"s3forcepathstyle"`
}

type LokiStorageConfig struct {
	Aws           LokiStorageConfigAws           `json:"aws"`
	BoltdbShipper LokiStorageConfigBoltdbShipper `json:"boltdb_shipper"`
}

type LokiCompactorConfig struct {
	// e.g. `/data/loki/boltdb-shipper-compactor`
	// WorkingDir string `json:"working_directory"`
	// e.g. `s3`
	SharedStore string `json:"shared_store"`
}

type LokiIngesterConfig struct {
	// default `3m`
	ChunkIdlePeriod string `json:"check_idle_period"`
	// default 262144
	CheckBlockSize int `json:"chunk_block_size"`
	// default: `1m`
	ChunkRetainPeriod string `json:"check_retain_period"`
}

type LokiTableManagerConfig struct {
	RetentionDeletesEnabled bool   `json:"retention_deletes_enabled"`
	RetentionPeriod         string `json:"retention_period"`
}

type LokiConfig struct {
	// Ingester      LokiIngesterConfig     `json:"ingester"`
	SchemaConfig  LokiConfigSchemaConfig  `json:"schema_config"`
	StorageConfig LokiStorageConfig       `json:"storage_config"`
	Compactor     LokiCompactorConfig     `json:"compactor"`
	TableManager  *LokiTableManagerConfig `json:"table_manager"`
}

type Loki struct {
	CommonConfig
	Image   Image       `json:"image"`
	Storage *Storage    `json:"persistence"`
	Config  *LokiConfig `json:"config"`
}

type PromtailVolumeHostPath struct {
	Path string `json:"path"`
}

type PromtailVolume struct {
	Name     string                 `json:"name"`
	HostPath PromtailVolumeHostPath `json:"hostPath"`
}

type PromtailVolumeMount struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
	ReadOnly  bool   `json:"readOnly"`
}

type Promtail struct {
	Enabled      bool                    `json:"enabled"`
	Resources    *api.HelmValueResources `json:"resources"`
	BusyboxImage string                  `json:"busyboxImage"`
	Image        Image                   `json:"image"`
	Volumes      []*PromtailVolume       `json:"volumes"`
	VolumeMounts []*PromtailVolumeMount  `json:"volumeMounts"`
}

type AdmissionWebhooksPatch struct {
	Enabled bool `json:"enabled"`
	// image: jettech/kube-webhook-certgen:v1.0.0
	Image Image `json:"image"`
}

type AdmissionWebhooks struct {
	Enabled bool                   `json:"enabled"`
	Patch   AdmissionWebhooksPatch `json:"patch"`
}

type PrometheusOperator struct {
	CommonConfig
	NodeSelector map[string]string `json:"nodeSelector"`
	// image: squareup/ghostunnel:v1.5.2
	TLSProxy          PromTLSProxy      `json:"tlsProxy"`
	AdmissionWebhooks AdmissionWebhooks `json:"admissionWebhooks"`
	// image: quay.io/coreos/prometheus-operator:v0.37.0
	Image Image `json:"image"`
	// image: quay.io/coreos/configmap-reload:v0.0.1
	ConfigmapReloadImage Image `json:"configmapReloadImage"`
	// image: quay.io/coreos/prometheus-config-reloader:v0.37.0
	PrometheusConfigReloaderImage Image `json:"prometheusConfigReloaderImage"`
	// image: k8s.gcr.io/hyperkube:v1.12.1
	HyperkubeImage Image `json:"hyperkubeImage"`
}

type MonitorStack struct {
	Prometheus             Prometheus             `json:"prometheus"`
	Alertmanager           Alertmanager           `json:"alertmanager"`
	NodeExporter           NodeExporter           `json:"nodeExporter"`
	PrometheusNodeExporter PrometheusNodeExporter `json:"prometheus-node-exporter"`
	KubeStateMetrics       KubeStateMetrics       `json:"kube-state-metrics"`
	Grafana                Grafana                `json:"grafana"`
	Loki                   Loki                   `json:"loki"`
	Promtail               Promtail               `json:"promtail"`
	PrometheusOperator     PrometheusOperator     `json:"prometheusOperator"`
}

type ThanosObjectStoreConfigConfig struct {
	api.ObjectStoreConfig
	SignatureVersion2 bool `json:"signature_version2"`
}

type ThanosObjectStoreConfig struct {
	Type   string                        `json:"type"`
	Config ThanosObjectStoreConfigConfig `json:"config"`
}

type ThanosQueryDnsDiscovery struct {
	Enabled           bool   `json:"enabled"`
	SidecarsService   string `json:"sidecarsService"`
	SidecarsNamespace string `json:"sidecarsNamespace"`
}

type ThanosQuery struct {
	CommonConfig

	Enabled      bool                    `json:"enabled"`
	DnsDiscovery ThanosQueryDnsDiscovery `json:"dnsDiscovery"`
	// Statically configure store APIs to connect with Thanos Query
	Stores []string `json:"stores"`
	// Number of Thanos Query replicas to deploy
	ReplicaCount int       `json:"replicaCount"`
	Resources    Resources `json:"resources"`
}

type Persistence struct {
	Enabled      bool     `json:"enabled"`
	StorageClass *string  `json:"storageClass"`
	AccessModes  []string `json:"accessModes"`
	Size         string   `json:"size"`
}

type ThanosStoregateway struct {
	CommonConfig

	Enabled     bool      `json:"enabled"`
	Resources   Resources `json:"resources"`
	Persistence Storage   `json:"persistence"`
}

type ThanosCompactor struct {
	CommonConfig

	Enabled     bool      `json:"enabled"`
	Resources   Resources `json:"resources"`
	Persistence Storage   `json:"persistence"`
}

type Thanos struct {
	Image          Image2                  `json:"image"`
	ClusterDomain  string                  `json:"clusterDomain"`
	ObjStoreConfig ThanosObjectStoreConfig `json:"objstoreConfig"`
	Query          ThanosQuery             `json:"query"`
	Storegateway   ThanosStoregateway      `json:"storegateway"`
	Compactor      ThanosCompactor         `json:"compactor"`
}

type MinioPersistence struct {
	Enabled      bool   `json:"enabled"`
	StorageClass string `json:"storageClass"`
	Size         string `json:"size"`
}

type CommonConfig struct {
	Enabled     bool                    `json:"enabled"`
	Tolerations []v1.Toleration         `json:"tolerations"`
	Affinity    *v1.Affinity            `json:"affinity"`
	Resources   *api.HelmValueResources `json:"resources"`
}

type Minio struct {
	CommonConfig

	Image Image `json:"image"`
	// ClusterDomain string `json:"clusterDomain"`
	// Minio client image
	McImage            Image  `json:"mcImage"`
	HelmKubectlJqImage Image  `json:"helmKubectlJqImage"`
	Mode               string `json:"mode"`
	// Number of MinIO containers running (applicable only for distributed mode)
	Replicas int `json:"replicas"`
	// Number of drives per node (applicable only for distributed mode)
	DrivesPerNode int `json:"drivesPerNode"`
	// Number of zones (applicable only for distributed mode)
	Zones int `json:"zones"`
	// Default MinIO admin accessKey
	AccessKey string `json:"accessKey"`
	// Default Minio admin secretKey
	SecretKey string `json:"secretKey"`
	// Default directory mount path, e.g. `/export`
	MountPath   string           `json:"mountPath"`
	Persistence MinioPersistence `json:"persistence"`
}

func GenerateHelmValues(config interface{}) map[string]interface{} {
	yamlStr := jsonutils.Marshal(config).YAMLString()
	vals := map[string]interface{}{}
	yaml.Unmarshal([]byte(yamlStr), &vals)
	return vals
}
