package templates

import (
	"encoding/base64"
	"time"
)

const KubeConfigProxyClientT = `apiVersion: v1
kind: Config
clusters:
- cluster:
    api-version: v1
    insecure-skip-tls-verify: true
    server: {{.KubernetesURL}}
  name: {{.ClusterName}}
contexts:
- context:
    cluster: {{.ClusterName}}
    user: {{.ComponentName}}
  name: "Default"
current-context: "Default"
users:
- name: {{.ComponentName}}
  user:
    client-certificate-data: {{.Crt}}
    client-key-data: {{.Key}}`

const KubeConfigClientT = `apiVersion: v1
kind: Config
clusters:
- cluster:
    api-version: v1
    certificate-authority-data: {{.Cacrt}}
    server: {{.KubernetesURL}}
  name: {{.ClusterName}}
contexts:
- context:
    cluster: {{.ClusterName}}
    user: {{.ComponentName}}
  name: "Default"
current-context: "Default"
users:
- name: {{.ComponentName}}
  user:
    client-certificate-data: {{.Crt}}
    client-key-data: {{.Key}}`

const KubeConfigTokenClientT = `apiVersion: v1
kind: Config
clusters:
- cluster:
    api-version: v1
    insecure-skip-tls-verify: true
    server: {{.KubernetesURL}}
  name: {{.ClusterName}}
contexts:
- context:
    cluster: {{.ClusterName}}
    user: {{.ComponentName}}
    namespace: {{.Namespace}}
  name: "Default"
current-context: "Default"
users:
- name: {{.ComponentName}}
  user:
    # expired at {{.Expired}}
    token: {{.Token}}`

func newKubeConfigMap(kubernetesURL, clusterName, componentName, cacrt, crt, key string) map[string]string {
	return map[string]string{
		"KubernetesURL": kubernetesURL,
		"ClusterName":   clusterName,
		"ComponentName": componentName,
		"Cacrt":         base64.StdEncoding.EncodeToString([]byte(cacrt)),
		"Crt":           base64.StdEncoding.EncodeToString([]byte(crt)),
		"Key":           base64.StdEncoding.EncodeToString([]byte(key)),
	}
}

func newKubeTokenConfigMap(kubernetesURL, clusterName, componentName, namespace, token string, expired time.Time) map[string]string {
	return map[string]string{
		"KubernetesURL": kubernetesURL,
		"ClusterName":   clusterName,
		"ComponentName": componentName,
		"Token":         token,
		"Namespace":     namespace,
		"Expired":       expired.String(),
	}
}

func GetKubeConfigByProxy(kubernetesURL, clusterName, componentName, cacrt, crt, key string) (string, error) {
	return CompileTemplateFromMap(KubeConfigProxyClientT, newKubeConfigMap(kubernetesURL, clusterName, componentName, cacrt, crt, key))
}

func GetKubeConfig(kubernetesURL, clusterName, componentName, cacrt, crt, key string) (string, error) {
	return CompileTemplateFromMap(KubeConfigClientT, newKubeConfigMap(kubernetesURL, clusterName, componentName, cacrt, crt, key))
}

func GetKubeTokenConfig(kubernetesURL, clusterName, componentName, namespace, token string, expired time.Time) (string, error) {
	return CompileTemplateFromMap(KubeConfigTokenClientT, newKubeTokenConfigMap(kubernetesURL, clusterName, componentName, namespace, token, expired))
}
