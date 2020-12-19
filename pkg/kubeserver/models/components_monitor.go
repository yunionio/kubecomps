package models

import (
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
	"k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/embed"
	"yunion.io/x/kubecomps/pkg/kubeserver/templates/components"
	"yunion.io/x/kubecomps/pkg/utils/certificates"
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
)

func init() {
	MonitorComponentManager = NewMonitorComponentManager()
	ComponentManager.RegisterDriver(newComponentDriverMonitor())
}

type SMonitorComponentManager struct {
	SComponentManager
	HelmComponentManager
}

type SMonitorComponent struct {
	SComponent
}

func NewMonitorComponentManager() *SMonitorComponentManager {
	man := new(SMonitorComponentManager)
	man.SComponentManager = *NewComponentManager(SMonitorComponent{},
		"kubecomponentmonitor",
		"kubecomponentmonitors")
	man.HelmComponentManager = *NewHelmComponentManager(MonitorNamespace, MonitorReleaseName, embed.MONITOR_STACK_8_12_13_TGZ)
	man.SetVirtualObject(man)
	return man
}

type componentDriverMonitor struct {
	baseComponentDriver
}

func newComponentDriverMonitor() IComponentDriver {
	return new(componentDriverMonitor)
}

func (c componentDriverMonitor) GetType() string {
	return api.ClusterComponentMonitor
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
	if err := c.validatePromtail(conf.Promtail); err != nil {
		return errors.Wrap(err, "component promtail")
	}
	return nil
}

func (c componentDriverMonitor) validateGrafana(userCred mcclient.TokenCredential, cluster *SCluster, conf *api.ComponentSettingMonitorGrafana) error {
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
	if conf.Storage.Enabled {
		if err := c.validateStorage(userCred, cluster, conf.Storage); err != nil {
			return err
		}
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
	if conf.Storage.Enabled {
		if err := c.validateStorage(userCred, cluster, conf.Storage); err != nil {
			return err
		}
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
	if err := MonitorComponentManager.EnsureNamespace(cluster, MonitorNamespace); err != nil {
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

func (c componentDriverMonitor) validatePromtail(conf *api.ComponentSettingMonitorPromtail) error {
	// TODO
	return nil
}

func (c componentDriverMonitor) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, input *api.ComponentUpdateInput) error {
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

// TODO: refactor this deduplicated code
func (c componentDriverMonitor) GetHelmValues(cluster *SCluster, setting *api.ComponentSettings) (map[string]interface{}, error) {
	return MonitorComponentManager.GetHelmValues(cluster, setting)
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
	imgRepo, err := cluster.GetImageRepository()
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
		rootUrl = fmt.Sprintf("%s/grafana/", rootUrl)
	}
	grafanaIni.Server.ServeFromSubPath = serveSubPath
	grafanaIni.Server.RootUrl = rootUrl

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
			Spec: components.PrometheusSpec{
				Image: mi("prometheus", "v2.15.2"),
			},
		},
		Alertmanager: components.Alertmanager{
			Spec: components.AlertmanagerSpec{
				Image: mi("alertmanager", "v0.20.0"),
			},
		},
		PrometheusNodeExporter: components.PrometheusNodeExporter{
			Image: mi("node-exporter", "v0.18.1"),
		},
		KubeStateMetrics: components.KubeStateMetrics{
			Image: mi("kube-state-metrics", "v1.9.4"),
		},
		Grafana: components.Grafana{
			AdminUser:     input.Grafana.AdminUser,
			AdminPassword: input.Grafana.AdminPassword,
			Sidecar: components.GrafanaSidecar{
				Image: mi("k8s-sidecar", "0.1.99"),
				Datasources: components.GrafanaSidecarDataSources{
					DefaultDatasourceEnabled: true,
				},
			},
			Image: mi("grafana", "6.7.1"),
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
			Image: mi("loki", "2.0.0"),
		},
		Promtail: components.Promtail{
			Image: mi("promtail", "2.0.0"),
		},
		PrometheusOperator: components.PrometheusOperator{
			Image:                         mi("prometheus-operator", "v0.37.0"),
			ConfigmapReloadImage:          mi("configmap-reload", "v0.0.1"),
			PrometheusConfigReloaderImage: mi("prometheus-config-reloader", "v0.37.0"),
			TLSProxy: components.PromTLSProxy{
				Image: mi("ghostunnel", "v1.5.2"),
			},
			AdmissionWebhooks: components.AdmissionWebhooks{
				Enabled: false,
				Patch: components.AdmissionWebhooksPatch{
					Enabled: false,
					Image:   mi("kube-webhook-certgen", "v1.0.0"),
				},
			},
		},
	}

	// inject prometheus spec
	if input.Prometheus.Storage != nil && input.Prometheus.Storage.Enabled {
		spec, err := components.NewPrometheusStorageSpec(*input.Prometheus.Storage)
		if err != nil {
			return nil, errors.Wrap(err, "prometheus storage spec")
		}
		conf.Prometheus.Spec.StorageSpec = spec
	}
	if input.Prometheus.ThanosSidecar != nil {
		image := mi("thanos", "v0.16.0")
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
	if input.Grafana.EnableThanosQueryDataSource {
		conf.Grafana.Sidecar.Datasources.DefaultDatasourceEnabled = false
		conf.Grafana.AdditionalDataSources = []components.GrafanaAdditionalDataSource{
			{
				Name:      "Thanos-Query",
				Type:      "prometheus",
				Url:       fmt.Sprintf("http://thanos-query.%s:9090", MonitorNamespace),
				Access:    "proxy",
				IsDefault: true,
			},
		}
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
		}
	}

	return components.GenerateHelmValues(conf), nil
}

func (m SMonitorComponentManager) CreateHelmResource(cluster *SCluster, setting *api.ComponentSettings) error {
	vals, err := m.GetHelmValues(cluster, setting)
	if err != nil {
		return errors.Wrap(err, "get helm config values")
	}
	return m.HelmComponentManager.CreateHelmResource(cluster, vals)
}

func (m SMonitorComponentManager) DeleteHelmResource(cluster *SCluster, setting *api.ComponentSettings) error {
	return m.HelmComponentManager.DeleteHelmResource(cluster)
}

func (m SMonitorComponentManager) UpdateHelmResource(cluster *SCluster, setting *api.ComponentSettings) error {
	vals, err := m.GetHelmValues(cluster, setting)
	if err != nil {
		return errors.Wrap(err, "get helm config values")
	}
	return m.HelmComponentManager.UpdateHelmResource(cluster, vals)
}
