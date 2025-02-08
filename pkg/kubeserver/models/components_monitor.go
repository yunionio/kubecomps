package models

import (
	"context"
	"fmt"
	"strings"

	"github.com/minio/minio-go/v7"
	"k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/embed"
	"yunion.io/x/kubecomps/pkg/kubeserver/options"
	"yunion.io/x/kubecomps/pkg/kubeserver/templates/components"
	"yunion.io/x/kubecomps/pkg/utils/certificates"
	"yunion.io/x/kubecomps/pkg/utils/grafana"
	"yunion.io/x/kubecomps/pkg/utils/objstore/s3"
)

var (
	MonitorComponentManager *SMonitorComponentManager
)

const (
	MonitorNamespace                  = "onecloud-monitoring"
	MonitorReleaseName                = "monitor"
	ThanosObjectStoreConfigSecretName = "thanos-objstore-config"
	ThanosObjectStoreConfigSecretKey  = "thanos.yaml"

	GrafanaSystemFolder = "Cloud-System"
	InfluxdbTelegrafDS  = "Influxdb-Telegraf"
	InfluxdbSystemDS    = "InfluxDB-system"
)

func init() {
	MonitorComponentManager = NewMonitorComponentManager()
	ComponentManager.RegisterDriver(newComponentDriverMonitor())
}

type SMonitorComponentManager struct {
	SComponentManager
}

type SMonitorComponent struct {
	SComponent
}

func NewMonitorComponentManager() *SMonitorComponentManager {
	man := new(SMonitorComponentManager)
	man.SComponentManager = *NewComponentManager(SMonitorComponent{},
		"kubecomponentmonitor",
		"kubecomponentmonitors")
	man.SetVirtualObject(man)
	return man
}

func (m *SMonitorComponentManager) GetComponentManager(cluster *SCluster) *HelmComponentManager {
	embedChart := embed.MONITOR_STACK_8_12_13_TGZ
	if strings.Contains(cluster.GetVersion(), "k3s") {
		embedChart = embed.MONITOR_STACK_V2_55_11_0_TGZ
	}
	return NewHelmComponentManager(MonitorNamespace, MonitorReleaseName, embedChart)
}

type componentDriverMonitor struct {
	helmComponentDriver
}

func newComponentDriverMonitor() IComponentDriver {
	return componentDriverMonitor{
		helmComponentDriver: newHelmComponentDriver(
			api.ClusterComponentMonitor,
			MonitorComponentManager,
		),
	}
}

func (c componentDriverMonitor) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, input *api.ComponentCreateInput) error {
	return c.validateSetting(ctx, userCred, cluster, input.Monitor)
}

func (c componentDriverMonitor) validateSetting(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, conf *api.ComponentSettingMonitor) error {
	if conf == nil {
		return httperrors.NewInputParameterError("monitor config is empty")
	}
	if err := c.validateGrafana(userCred, cluster, conf.Grafana); err != nil {
		return errors.Wrap(err, "component grafana")
	}
	if err := c.validateLoki(ctx, userCred, cluster, conf.Loki); err != nil {
		return errors.Wrap(err, "component loki")
	}
	if err := c.validatePrometheus(ctx, userCred, cluster, conf.Prometheus); err != nil {
		return errors.Wrap(err, "component prometheus")
	}
	promtailConf, err := c.validatePromtail(cluster, conf.Loki, conf.Promtail)
	if err != nil {
		return errors.Wrap(err, "component promtail")
	}
	conf.Promtail = promtailConf
	return nil
}

func (c componentDriverMonitor) validateGrafana(userCred mcclient.TokenCredential, cluster *SCluster, conf *api.ComponentSettingMonitorGrafana) error {
	if conf.Disable {
		return nil
	}
	var err error
	conf.Resources, err = c.setDefaultHelmValueResources(
		conf.Resources,
		api.NewHelmValueResource("1", "1024Mi"),
		api.NewHelmValueResource("0.01", "10Mi"),
	)
	if err != nil {
		return err
	}
	if conf.Storage.Enabled {
		if err := c.validateStorage(userCred, cluster, conf.Storage); err != nil {
			return err
		}
	}
	if conf.Host == "" && conf.PublicAddress == "" {
		return httperrors.NewInputParameterError("grafana public address or host must provide")
	}
	if conf.TLSKeyPair == nil {
		// return httperrors.NewInputParameterError("grafana tls key pair must provide")
		kp, err := certificates.GetOrGenerateCACert(nil, "grafana-tls")
		if err != nil {
			return errors.Wrap(err, "generate grafana tls keypair")
		}
		conf.TLSKeyPair = &api.TLSKeyPair{
			Certificate: string(kp.Cert),
			Key:         string(kp.Key),
		}
	}
	if err := c.validateGrafanaTLSKeyPair(conf.TLSKeyPair); err != nil {
		return errors.Wrap(err, "validate tls key pair")
	}
	return nil
}

func (c componentDriverMonitor) validateGrafanaTLSKeyPair(pair *api.TLSKeyPair) error {
	if pair.Certificate == "" {
		return httperrors.NewInputParameterError("tls certificate not provide")
	}
	if pair.Key == "" {
		return httperrors.NewInputParameterError("tls key not provide")
	}
	if pair.Name == "" {
		pair.Name = "grafana-ingress-tls"
	}
	return nil
}

func validateObjectStore(ctx context.Context, conf *api.ObjectStoreConfig) error {
	for key, val := range map[string]string{
		"bucket":     conf.Bucket,
		"endpoint":   conf.Endpoint,
		"access key": conf.AccessKey,
		"secret key": conf.SecretKey,
	} {
		if val == "" {
			return httperrors.NewNotEmptyError("%s is not provide", key)
		}
	}

	if err := s3.CheckValidBucketNameStrict(conf.Bucket); err != nil {
		return httperrors.NewInputParameterError("bucket name %q is invaild: %s", conf.Bucket, err)
	}

	cli, err := s3.NewClient(&s3.Config{
		Endpoint:  conf.Endpoint,
		Secure:    !conf.Insecure,
		AccessKey: conf.AccessKey,
		SecretKey: conf.SecretKey,
	})
	if err != nil {
		return err
	}

	exists, err := cli.BucketExists(ctx, conf.Bucket)
	if err != nil {
		return errors.Wrap(err, "check bucket exists")
	}
	if !exists {
		// return httperrors.NewNotFoundError("bucket %s not found", conf.Bucket)
		if err := cli.MakeBucket(ctx, conf.Bucket, minio.MakeBucketOptions{}); err != nil {
			return errors.Wrapf(err, "make bucket %s", conf.Bucket)
		}
	}

	return nil
}

func (c componentDriverMonitor) validateLoki(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, conf *api.ComponentSettingMonitorLoki) error {
	if conf.Disable {
		return nil
	}
	if conf.Storage.Enabled {
		if err := c.validateStorage(userCred, cluster, conf.Storage); err != nil {
			return err
		}
	}
	var err error
	conf.Resources, err = c.setDefaultHelmValueResources(
		conf.Resources,
		api.NewHelmValueResource("2", "2048Mi"),
		api.NewHelmValueResource("0.01", "10Mi"),
	)
	if err != nil {
		return err
	}

	if conf.ObjectStoreConfig != nil {
		if err := validateObjectStore(ctx, conf.ObjectStoreConfig); err != nil {
			return err
		}
	}
	return nil
}

func (c componentDriverMonitor) validatePrometheus(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, conf *api.ComponentSettingMonitorPrometheus) error {
	if conf == nil {
		return httperrors.NewInputParameterError("config is empty")
	}
	if conf.Disable {
		return nil
	}
	if conf.Storage.Enabled {
		if err := c.validateStorage(userCred, cluster, conf.Storage); err != nil {
			return err
		}
	}
	var err error
	conf.Resources, err = c.setDefaultHelmValueResources(
		conf.Resources,
		api.NewHelmValueResource("2", "2048Mi"),
		api.NewHelmValueResource("0.01", "10Mi"),
	)
	if err != nil {
		return err
	}
	if conf.ThanosSidecar != nil {
		if err := c.validatePrometheusThanos(ctx, cluster, conf.ThanosSidecar); err != nil {
			return err
		}
	}
	return nil
}

func (c componentDriverMonitor) createOrUpdateThanosObjectStoreSecret(ctx context.Context, cluster *SCluster, conf *components.ThanosObjectStoreConfig) error {
	yamlStr := jsonutils.Marshal(conf).YAMLString()

	cli, err := cluster.GetRemoteClient()
	if err != nil {
		return errors.Wrapf(err, "get cluster %s remote client", cluster.GetName())
	}
	if err := MonitorComponentManager.GetComponentManager(cluster).EnsureNamespace(cluster, MonitorNamespace); err != nil {
		return errors.Wrap(err, "ensure namespace")
	}
	secrets := cli.GetClientset().CoreV1().Secrets(MonitorNamespace)
	obj, err := secrets.Get(ctx, ThanosObjectStoreConfigSecretName, metav1.GetOptions{})
	if err != nil {
		if !kerrors.IsNotFound(err) {
			return errors.Wrapf(err, "get remote secret %s/%s", MonitorNamespace, ThanosObjectStoreConfigSecretName)
		}
		// create it
		secretObj := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: ThanosObjectStoreConfigSecretName,
			},
			StringData: map[string]string{
				ThanosObjectStoreConfigSecretKey: yamlStr,
			},
		}
		if _, err := secrets.Create(ctx, secretObj, metav1.CreateOptions{}); err != nil {
			return errors.Wrapf(err, "create remote secret %s/%s", MonitorNamespace, ThanosObjectStoreConfigSecretName)
		}
		return nil
	}

	// update it
	newObj := obj.DeepCopy()
	if newObj.StringData == nil {
		newObj.StringData = make(map[string]string)
	}
	newObj.StringData[ThanosObjectStoreConfigSecretKey] = yamlStr
	if _, err := secrets.Update(ctx, newObj, metav1.UpdateOptions{}); err != nil {
		return errors.Wrapf(err, "update remote secret %s/%s", MonitorNamespace, ThanosObjectStoreConfigSecretKey)
	}
	return nil
}

func (c componentDriverMonitor) validatePrometheusThanos(ctx context.Context, cluster *SCluster, conf *api.ComponentSettingMonitorPrometheusThanos) error {
	if err := validateObjectStore(ctx, conf.ObjectStoreConfig); err != nil {
		return errors.Wrap(err, "validate object store")
	}

	thanosConf := &components.ThanosObjectStoreConfig{
		Type: "s3",
		Config: components.ThanosObjectStoreConfigConfig{
			ObjectStoreConfig: *conf.ObjectStoreConfig,
			SignatureVersion2: true,
		},
	}
	if err := c.createOrUpdateThanosObjectStoreSecret(ctx, cluster, thanosConf); err != nil {
		return errors.Wrap(err, "create or update thanos object store secret")
	}

	return nil
}

func (c componentDriverMonitor) validatePromtail(cluster *SCluster, lokiConf *api.ComponentSettingMonitorLoki, conf *api.ComponentSettingMonitorPromtail) (*api.ComponentSettingMonitorPromtail, error) {
	// TODO
	if conf == nil {
		conf = &api.ComponentSettingMonitorPromtail{
			Disable: lokiConf.Disable,
		}

	}
	if conf.Disable {
		return conf, nil
	}

	defaultDockerPath := "/opt/docker/containers"
	if !cluster.IsSystemCluster() {
		defaultDockerPath = "/var/lib/docker/containers"
	}
	conf.DockerVolumeMount = &api.ComponentSettingVolume{
		HostPath:  defaultDockerPath,
		MountPath: defaultDockerPath,
	}
	defaultPodPath := "/var/log/pods"
	conf.PodsVolumeMount = &api.ComponentSettingVolume{
		HostPath:  defaultPodPath,
		MountPath: defaultPodPath,
	}

	var err error
	conf.Resources, err = c.setDefaultHelmValueResources(
		conf.Resources,
		api.NewHelmValueResource("1", "1024Mi"),
		api.NewHelmValueResource("0.01", "10Mi"),
	)
	if err != nil {
		return nil, err
	}
	return conf, nil
}

func (c componentDriverMonitor) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, input *api.ComponentUpdateInput) error {
	comp, err := cluster.GetComponentByType(input.Type)
	if err != nil {
		return err
	}
	oldSetting, _ := comp.GetSettings()
	if oldSetting != nil {
		if input.Monitor.Grafana.TLSKeyPair == nil {
			input.Monitor.Grafana.TLSKeyPair = oldSetting.Monitor.Grafana.TLSKeyPair
		}
	}
	return c.validateSetting(ctx, userCred, cluster, input.Monitor)
}

func (c componentDriverMonitor) GetCreateSettings(input *api.ComponentCreateInput) (*api.ComponentSettings, error) {
	if input.ComponentSettings.Namespace == "" {
		input.ComponentSettings.Namespace = MonitorNamespace
	}
	return &input.ComponentSettings, nil
}

func (c componentDriverMonitor) GetUpdateSettings(oldSetting *api.ComponentSettings, input *api.ComponentUpdateInput) (*api.ComponentSettings, error) {
	oldSetting.Monitor = input.Monitor
	return oldSetting, nil
}

func (c componentDriverMonitor) DoEnable(cluster *SCluster, setting *api.ComponentSettings) error {
	return MonitorComponentManager.CreateHelmResource(cluster, setting)
}

func (c componentDriverMonitor) DoDisable(cluster *SCluster, setting *api.ComponentSettings) error {
	return MonitorComponentManager.DeleteHelmResource(cluster, setting)
}

func (c componentDriverMonitor) DoUpdate(cluster *SCluster, setting *api.ComponentSettings) error {
	return MonitorComponentManager.UpdateHelmResource(cluster, setting)
}

func (c componentDriverMonitor) FetchStatus(cluster *SCluster, comp *SComponent, status *api.ComponentsStatus) error {
	if status.Monitor == nil {
		status.Monitor = new(api.ComponentStatusMonitor)
	}
	c.InitStatus(comp, &status.Monitor.ComponentStatus)
	return nil
}

func (m SMonitorComponentManager) GetHelmValues(cluster *SCluster, setting *api.ComponentSettings) (map[string]interface{}, error) {
	imgRepo, err := m.GetImageRepository(cluster, setting)
	if err != nil {
		return nil, errors.Wrapf(err, "get cluster %s repo", cluster.GetName())
	}
	input := setting.Monitor
	if input.Grafana.AdminUser == "" {
		input.Grafana.AdminUser = "admin"
	}
	if input.Grafana.AdminPassword == "" {
		input.Grafana.AdminPassword = "prom-operator"
	}
	repo := imgRepo.Url
	mi := func(name, tag string) components.Image {
		return components.Image{
			Repository: fmt.Sprintf("%s/%s", repo, name),
			Tag:        tag,
		}
	}
	grafanaHost := input.Grafana.Host
	if grafanaHost == "" {
		grafanaHost = input.Grafana.PublicAddress
	}

	grafanaProto := "https"
	rootUrl := fmt.Sprintf("%s://%s", grafanaProto, grafanaHost)
	serveSubPath := false
	grafanaIni := &components.GrafanaIni{
		Server: &components.GrafanaIniServer{},
	}
	if !input.Grafana.DisableSubpath {
		serveSubPath = true
		subpath := input.Grafana.Subpath
		if subpath == "" {
			subpath = "grafana"
		}
		rootUrl = fmt.Sprintf("%s/%s/", rootUrl, subpath)
	}
	grafanaIni.Server.ServeFromSubPath = serveSubPath
	grafanaIni.Server.RootUrl = rootUrl
	if input.Grafana.EnforceDomain {
		grafanaIni.Server.Domain = grafanaHost
		grafanaIni.Server.EnforceDomain = true
	}

	grafanaIni.AnonyousAuth = &components.GrafanaIniAnonyousAuth{
		Enabled: false,
		OrgName: "Main Org.",
		OrgRole: "Viewer",
	}

	if input.Grafana.OAuth != nil {
		oauth := input.Grafana.OAuth
		grafanaIni.OAuth = &components.GrafanaIniOAuth{
			Enabled:           oauth.Enabled,
			ClientId:          oauth.ClientId,
			ClientSecret:      oauth.ClientSecret,
			Scopes:            oauth.Scopes,
			AuthURL:           oauth.AuthURL,
			TokenURL:          oauth.TokenURL,
			APIURL:            oauth.APIURL,
			AllowedDomains:    oauth.AllowedDomains,
			AllowSignUp:       oauth.AllowSignUp,
			RoleAttributePath: oauth.RoleAttributePath,
		}
	}

	if input.Grafana.DB != nil {
		db := input.Grafana.DB
		if db.Host == "" {
			return nil, errors.Errorf("grafana db host is empty")
		}
		if db.Database == "" {
			return nil, errors.Errorf("grafana db database is empty")
		}
		if db.Username == "" {
			return nil, errors.Errorf("grafana db username is empty")
		}
		if db.Password == "" {
			return nil, errors.Errorf("grafana db password is empty")
		}
		if db.Port == 0 {
			db.Port = 3306
		}
		grafanaIni.Database = &components.GrafanaIniDatabase{
			Type:     "mysql",
			Host:     fmt.Sprintf("%s:%d", db.Host, db.Port),
			Name:     db.Database,
			User:     db.Username,
			Password: db.Password,
		}
	}

	if input.Grafana.TLSKeyPair == nil {
		return nil, errors.Errorf("grafana tls key pair not provided")
	}
	grafanaIngressTLS := []*api.IngressTLS{
		{
			SecretName: input.Grafana.TLSKeyPair.Name,
		},
	}
	conf := components.MonitorStack{
		Prometheus: components.Prometheus{
			Enabled: !input.Prometheus.Disable,
			Spec: components.PrometheusSpec{
				CommonConfig: components.CommonConfig{
					Enabled:   !input.Prometheus.Disable,
					Resources: input.Prometheus.Resources,
				},
				Image: mi("prometheus", "v2.28.1"),
			},
		},
		Alertmanager: components.Alertmanager{
			Enabled: !input.Prometheus.Disable,
			Spec: components.AlertmanagerSpec{
				CommonConfig: components.CommonConfig{
					Enabled:   !input.Prometheus.Disable,
					Resources: input.Prometheus.Resources,
				},
				Image: mi("alertmanager", "v0.22.2"),
			},
		},
		NodeExporter: components.NodeExporter{
			Enabled: !input.Prometheus.Disable,
		},
		PrometheusNodeExporter: components.PrometheusNodeExporter{
			Enabled: !input.Prometheus.Disable,
			Image:   mi("node-exporter", "v1.2.0"),
		},
		KubeStateMetrics: components.KubeStateMetrics{
			CommonConfig: components.CommonConfig{
				Enabled:   !input.Prometheus.Disable,
				Resources: input.Prometheus.Resources,
			},
			Image: mi("kube-state-metrics", "v1.9.8"),
		},
		Grafana: components.Grafana{
			CommonConfig: components.CommonConfig{
				Enabled:   !input.Grafana.Disable,
				Resources: input.Grafana.Resources,
			},
			AdminUser:     input.Grafana.AdminUser,
			AdminPassword: input.Grafana.AdminPassword,
			Sidecar: components.GrafanaSidecar{
				Image: mi("k8s-sidecar", "1.12.2"),
				Dashboards: components.GrafanaSidecarDashboards{
					Enabled: !input.Prometheus.Disable,
				},
				Datasources: components.GrafanaSidecarDataSources{
					DefaultDatasourceEnabled: true,
				},
			},
			Image: mi("grafana", "7.5.17"),
			Service: &components.Service{
				Type: string(v1.ServiceTypeClusterIP),
			},
			Ingress: &components.GrafanaIngress{
				Enabled: true,
				Host:    input.Grafana.Host,
				Secret:  input.Grafana.TLSKeyPair,
				TLS:     grafanaIngressTLS,
			},
			GrafanaIni: grafanaIni,
		},
		Loki: components.Loki{
			CommonConfig: components.CommonConfig{
				Enabled:   !input.Loki.Disable,
				Resources: input.Loki.Resources,
			},
			Image: mi("loki", "2.2.1"),
		},
		Promtail: components.Promtail{
			Enabled:   !input.Loki.Disable,
			Resources: input.Promtail.Resources,
			Image:     mi("promtail", "2.2.1"),
		},
	}

	conf.PrometheusOperator = m.getPrometheusOperatorConf(cluster, input, repo)

	// inject prometheus spec
	if input.Prometheus.Storage != nil && input.Prometheus.Storage.Enabled {
		spec, err := components.NewPrometheusStorageSpec(*input.Prometheus.Storage)
		if err != nil {
			return nil, errors.Wrap(err, "prometheus storage spec")
		}
		conf.Prometheus.Spec.StorageSpec = spec
	}
	if input.Prometheus.ThanosSidecar != nil {
		image := mi("thanos", "v0.22.0")
		conf.Prometheus.Spec.Thanos = components.ThanosSidecarSpec{
			BaseImage: image.Repository,
			Version:   image.Tag,
			ObjectStorageConfig: components.ObjectStorageConfig{
				Name: ThanosObjectStoreConfigSecretName,
				Key:  ThanosObjectStoreConfigSecretKey,
			},
		}
		conf.Prometheus.Spec.Retention = "4h"
	}

	// inject grafana spec
	if input.Grafana.Storage != nil && input.Grafana.Storage.Enabled {
		spec, err := components.NewPVCStorage(input.Grafana.Storage)
		if err != nil {
			return nil, errors.Wrap(err, "grafana storage spec")
		}
		conf.Grafana.Storage = spec
	}
	if !input.Grafana.DisableSubpath {
		conf.Grafana.Ingress.Path = "/grafana"
	}
	if input.Grafana.EnableThanosQueryDataSource && !input.Prometheus.Disable {
		conf.Grafana.Sidecar.Datasources.DefaultDatasourceEnabled = false
		conf.Grafana.AdditionalDataSources = []components.GrafanaAdditionalDataSource{
			{
				Name:      "Prometheus",
				Type:      "prometheus",
				Url:       fmt.Sprintf("http://monitor-monitor-stack-prometheus.%s:9090", MonitorNamespace),
				Access:    "proxy",
				IsDefault: true,
			},
			{
				Name:   "Thanos-Query",
				Type:   "prometheus",
				Url:    fmt.Sprintf("http://thanos-query.%s:9090", MonitorNamespace),
				Access: "proxy",
			},
		}
	}
	if input.Prometheus.Disable {
		conf.Grafana.Sidecar.Datasources.DefaultDatasourceEnabled = false
		conf.Grafana.AdditionalDataSources = []components.GrafanaAdditionalDataSource{}
		conf.Grafana.Sidecar.Dashboards.Enabled = false
	}

	if cluster.IsSystemCluster() {
		conf.Grafana.AdditionalDataSources = append(conf.Grafana.AdditionalDataSources,
			components.GrafanaAdditionalDataSource{
				Name:     InfluxdbTelegrafDS,
				Type:     "influxdb",
				Access:   "proxy",
				Database: "telegraf",
				Url:      fmt.Sprintf("https://default-influxdb.onecloud:30086"),
				JsonData: &components.GrafanaDataSourceJsonData{
					TlsSkipVerify: true,
				},
			},
			components.GrafanaAdditionalDataSource{
				Name:     InfluxdbSystemDS,
				Type:     "influxdb",
				Access:   "proxy",
				Database: "system",
				Url:      fmt.Sprintf("https://default-influxdb.onecloud:30086"),
				JsonData: &components.GrafanaDataSourceJsonData{
					TlsSkipVerify: true,
				},
			},
			components.GrafanaAdditionalDataSource{
				Name:   "Loki",
				Type:   "loki",
				Access: "proxy",
				Url:    fmt.Sprintf(fmt.Sprintf("http://%s-loki:3100", MonitorReleaseName)),
			},
		)
	}

	// inject loki spec
	if input.Loki.Storage != nil && input.Loki.Storage.Enabled {
		spec, err := components.NewPVCStorage(input.Loki.Storage)
		if err != nil {
			return nil, errors.Wrap(err, "loki storage")
		}
		conf.Loki.Storage = spec
	}
	if input.Loki.ObjectStoreConfig != nil {
		objConf := input.Loki.ObjectStoreConfig
		conf.Loki.Config = &components.LokiConfig{
			SchemaConfig: components.LokiConfigSchemaConfig{
				Configs: []components.LokiSchemaConfig{
					{
						From:        "2020-10-24",
						Store:       "boltdb-shipper",
						ObjectStore: "aws",
						Schema:      "v11",
						Index: components.LokiSchemaConfigIndex{
							Prefix: "index_",
							Period: "24h",
						},
					},
				},
			},
			StorageConfig: components.LokiStorageConfig{
				Aws: components.LokiStorageConfigAws{
					S3ForcepathStyle: true,
					S3:               fmt.Sprintf("s3://%s:%s@%s/%s", objConf.AccessKey, objConf.SecretKey, objConf.Endpoint, objConf.Bucket),
				},
				BoltdbShipper: components.LokiStorageConfigBoltdbShipper{
					// ActiveIndexDirectory: "/data/loki/boltdb-shipper-active",
					// CacheLocation:        "/data/loki/boltdb-shipper-cache",
					CacheTTL:    "24h",
					SharedStore: "s3",
				},
			},
			Compactor: components.LokiCompactorConfig{
				// WorkingDir:  "/data/loki/boltdb-shipper-compactor",
				SharedStore: "s3",
			},
			TableManager: &components.LokiTableManagerConfig{
				RetentionDeletesEnabled: true,
				// 7 days
				RetentionPeriod: "168h",
			},
		}
	}

	// inject promtail spec
	if !input.Loki.Disable {
		conf.Promtail.Volumes = []*components.PromtailVolume{
			{
				Name: "docker",
				HostPath: components.PromtailVolumeHostPath{
					Path: input.Promtail.DockerVolumeMount.HostPath,
				},
			},
			{
				Name: "pods",
				HostPath: components.PromtailVolumeHostPath{
					Path: input.Promtail.PodsVolumeMount.HostPath,
				},
			},
		}
		conf.Promtail.VolumeMounts = []*components.PromtailVolumeMount{
			{
				Name:      "docker",
				MountPath: input.Promtail.DockerVolumeMount.MountPath,
				ReadOnly:  true,
			},
			{
				Name:      "pods",
				MountPath: input.Promtail.PodsVolumeMount.MountPath,
				ReadOnly:  true,
			},
		}
	}

	// set system cluster common config
	if cluster.IsSystemCluster() {
		conf.Grafana.CommonConfig = getSystemComponentCommonConfig(
			conf.Grafana.CommonConfig,
			false, input.Grafana.Disable)
		conf.Loki.CommonConfig = getSystemComponentCommonConfig(
			conf.Loki.CommonConfig,
			false, input.Loki.Disable)
		conf.Prometheus.Spec.CommonConfig = getSystemComponentCommonConfig(
			conf.Prometheus.Spec.CommonConfig,
			false, input.Prometheus.Disable)
		conf.Alertmanager.Spec.CommonConfig = getSystemComponentCommonConfig(
			conf.Alertmanager.Spec.CommonConfig,
			false, input.Prometheus.Disable)
		conf.PrometheusOperator.CommonConfig = getSystemComponentCommonConfig(
			conf.PrometheusOperator.CommonConfig,
			false, false)
		conf.KubeStateMetrics.CommonConfig = getSystemComponentCommonConfig(
			conf.KubeStateMetrics.CommonConfig,
			false, input.Prometheus.Disable)
	}

	// disable resource management
	if setting.DisableResourceManagement {
		conf.Prometheus.Spec.Resources = components.Resources{}
		conf.Alertmanager.Spec.Resources = nil
		conf.KubeStateMetrics.Resources = nil
		conf.Grafana.Resources = nil
		conf.Loki.Resources = nil
		conf.Promtail.Resources = nil
		conf.PrometheusOperator.Resources = nil
	}

	return components.GenerateHelmValues(conf), nil
}

func (m SMonitorComponentManager) CreateHelmResource(cluster *SCluster, setting *api.ComponentSettings) error {
	vals, err := m.GetHelmValues(cluster, setting)
	if err != nil {
		return errors.Wrap(err, "get helm config values")
	}
	return m.GetComponentManager(cluster).CreateHelmResource(cluster, vals)
}

func (m SMonitorComponentManager) DeleteHelmResource(cluster *SCluster, setting *api.ComponentSettings) error {
	return m.GetComponentManager(cluster).DeleteHelmResource(cluster)
}

func (m SMonitorComponentManager) UpdateHelmResource(cluster *SCluster, setting *api.ComponentSettings) error {
	vals, err := m.GetHelmValues(cluster, setting)
	if err != nil {
		return errors.Wrap(err, "get helm config values")
	}
	return m.GetComponentManager(cluster).UpdateHelmResource(cluster, vals)
}

func (m SMonitorComponentManager) SyncSystemGrafanaDashboard(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	if err := m.syncSystemGrafanaDashboard(ctx); err != nil {
		log.Errorf("Sync system grafana dashboard error: %v", err)
		return
	}
	log.Infof("System telegraf dashboard to grafana synced")
}

func (m SMonitorComponentManager) syncSystemGrafanaDashboard(ctx context.Context) error {
	sysCls, err := ClusterManager.GetSystemCluster()
	if err != nil {
		return errors.Wrap(err, "get system cluster")
	}
	if sysCls == nil {
		return errors.Wrap(err, "system cluster is not exists")
	}

	comp, err := sysCls.GetComponentByType(api.ClusterComponentMonitor)
	if err != nil {
		return errors.Wrap(err, "get monitor component")
	}

	if comp == nil {
		return nil
	}

	settings, err := comp.GetSettings()
	if err != nil {
		return errors.Wrap(err, "get component settings")
	}

	setting := settings.Monitor
	if setting == nil {
		return errors.Wrap(err, "monitor setting is nil")
	}

	gs := setting.Grafana
	if gs == nil {
		return errors.Wrap(err, "grafana setting is nil")
	}

	if gs.Disable {
		return nil
	}

	// FIX: hard code
	defaultDBInputs := []grafana.ImportDashboardInput{
		{
			Name:     "DS_LINUXSERVER",
			PluginId: "influxdb",
			Type:     "datasource",
			Value:    InfluxdbTelegrafDS,
		},
	}

	apiUrl := fmt.Sprintf("http://monitor-grafana.%s", MonitorNamespace)
	if gs.Host != "" {
		apiUrl = fmt.Sprintf("https://%s", gs.Host)
		if !gs.DisableSubpath && gs.Subpath != "" {
			apiUrl = fmt.Sprintf("%s/%s", apiUrl, gs.Subpath)
		}
	}
	cli := grafana.NewClient(apiUrl, gs.AdminUser, gs.AdminPassword).
		SetDebug(options.Options.LogLevel == "debug")

	// ensure system folder
	/*
	 * folders, err := cli.ListFolders(ctx)
	 * if err != nil {
	 * 	return errors.Wrap(err, "list grafana folders")
	 * }
	 * var sysFolder *grafana.FolderHit
	 * for _, f := range folders {
	 * 	if f.Title == GrafanaSystemFolder {
	 * 		tmp := f
	 * 		sysFolder = &tmp
	 * 		break
	 * 	}
	 * }
	 * if sysFolder == nil {
	 * 	// create folder
	 * 	f, err := cli.CreateFolder(ctx, grafana.CreateFolderParams{
	 * 		Title: GrafanaSystemFolder,
	 * 	})
	 * 	if err != nil {
	 * 		return errors.Wrap(err, "create system folder")
	 * 	}
	 * 	log.Errorf("===create folders %#v", f)
	 * 	sysFolder = &f.FolderHit
	 * }
	 */

	if err := cli.ImportDashboard(ctx,
		embed.Get(embed.LINUX_SERVER_REV1_JSON),
		grafana.ImportDashboardParams{
			// FolderId:  sysFolder.Id,
			FolderId:  0,
			Overwrite: true,
			Inputs:    defaultDBInputs,
		},
	); err != nil {
		return errors.Wrap(err, "import telegraf system dashboard to grafana")
	}

	log.Infof("import telegraf system dashboard to grafana successful")

	if err := cli.ImportDashboard(ctx,
		embed.Get(embed.SERVICE_MONITOR_JSON),
		grafana.ImportDashboardParams{
			FolderId:  0,
			Overwrite: true,
			Inputs:    nil,
		},
	); err != nil {
		return errors.Wrap(err, "import service monitor dashboard to grafana")
	}

	log.Infof("import service monitor dashboard to grafana successful")

	return nil
}

func (m *SMonitorComponentManager) getPrometheusOperatorConf(cluster *SCluster, input *api.ComponentSettingMonitor, repo string) components.PrometheusOperator {
	mi := func(name, tag string) components.Image {
		return components.Image{
			Repository: fmt.Sprintf("%s/%s", repo, name),
			Tag:        tag,
		}
	}
	mi2 := func(name, tag string) components.Image2 {
		return components.Image2{
			Registry:   repo,
			Repository: name,
			Tag:        tag,
		}
	}
	return components.PrometheusOperator{
		CommonConfig: components.CommonConfig{
			// must enable to control prometheus lifecycle
			Enabled:   true,
			Resources: input.Prometheus.Resources,
		},
		Image:                         mi2("prometheus-operator", ""),
		ConfigmapReloadImage:          mi("configmap-reload", "v0.5.0"),
		PrometheusConfigReloaderImage: mi("prometheus-config-reloader", "v0.38.1"),
		TLSProxy: components.PromTLSProxy{
			Image: mi("ghostunnel", "v1.5.3"),
		},
		TLS: components.PrometheusOperatorTLS{
			Enabled: false,
		},
		AdmissionWebhooks: components.AdmissionWebhooks{
			Enabled: false,
			Patch: components.AdmissionWebhooksPatch{
				Enabled: false,
				Image:   mi("kube-webhook-certgen", "v1.5.2"),
			},
		},
	}
}
