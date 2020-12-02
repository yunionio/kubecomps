package userdata

import (
	"bytes"
	"text/template"

	kubeproxyconfigv1alpha1 "k8s.io/kube-proxy/config/v1alpha1"
	kubeadmv1beta1 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1beta1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/machines/kubeadm"
)

const (
	InitScript = `
#!/usr/bin/env bash
set -o verbose
set -o errexit
set -o nounset
set -o pipefail

mkdir -p /etc/docker

cat >/etc/docker/daemon.json <<EOF
{{.DockerConfigJSON}}
EOF

systemctl enable docker ntpd
systemctl restart docker ntpd
hwclock --utc --hctosys

{{ if or .InitConfiguration .ControlJoinConfiguration }}
mkdir -p /etc/kubernetes/pki/etcd

echo '{{.CACert}}' > /etc/kubernetes/pki/ca.crt
echo '{{.CAKey}}' > /etc/kubernetes/pki/ca.key
echo '{{.EtcdCACert}}' > /etc/kubernetes/pki/etcd/ca.crt
echo '{{.EtcdCAKey}}' > /etc/kubernetes/pki/etcd/ca.key
echo '{{.FrontProxyCACert}}' > /etc/kubernetes/pki/front-proxy-ca.crt
echo '{{.FrontProxyCAKey}}' > /etc/kubernetes/pki/front-proxy-ca.key
echo '{{.SaCert}}' > /etc/kubernetes/pki/sa.pub
echo '{{.SaKey}}' > /etc/kubernetes/pki/sa.key
{{ end }}

{{ if .InitConfiguration }}
cat >/tmp/kubeadm-init.yaml <<EOF
---
{{.InitConfiguration}}
---
{{.ClusterConfiguration}}
---
{{.KubeProxyConfiguration}}
---
EOF
kubeadm init --config /tmp/kubeadm-init.yaml

{{ else if .ControlJoinConfiguration }}
cat >/tmp/kubeadm-join.yaml <<EOF
{{.ControlJoinConfiguration}}
EOF
kubeadm join --config /tmp/kubeadm-join.yaml

{{ else }}
cat >/tmp/kubeadm-node.yaml <<EOF
{{.NodeJoinConfiguration}}
EOF
kubeadm join --config /tmp/kubeadm-node.yaml
{{ end }}

systemctl enable kubelet
`
)

type InitScriptConfig struct {
	DockerConfigJSON         string
	InitConfiguration        string
	ClusterConfiguration     string
	KubeProxyConfiguration   string
	ControlJoinConfiguration string
	NodeJoinConfiguration    string

	CACert           string
	CAKey            string
	EtcdCACert       string
	EtcdCAKey        string
	FrontProxyCACert string
	FrontProxyCAKey  string
	SaCert           string
	SaKey            string
}

func (c InitScriptConfig) ToScript() (string, error) {
	t, err := template.New("script").Parse(InitScript)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse script template")
	}
	var out bytes.Buffer
	if err := t.Execute(&out, c); err != nil {
		return "", errors.Wrapf(err, "failed to execute script template")
	}
	return out.String(), nil
}

type InitNodeConfig struct {
	DockerConfiguration    *api.DockerConfig
	InitConfiguration      *kubeadmv1beta1.InitConfiguration
	ClusterConfiguration   *kubeadmv1beta1.ClusterConfiguration
	KubeProxyConfiguration *kubeproxyconfigv1alpha1.KubeProxyConfiguration

	CACert           string
	CAKey            string
	EtcdCACert       string
	EtcdCAKey        string
	FrontProxyCACert string
	FrontProxyCAKey  string
	SaCert           string
	SaKey            string
}

func (c InitNodeConfig) ToScript() (string, error) {
	initYAML, err := kubeadm.ConfigurationToYAML(c.InitConfiguration)
	if err != nil {
		return "", err
	}
	clusterYAML, err := kubeadm.ConfigurationToYAML(c.ClusterConfiguration)
	if err != nil {
		return "", err
	}
	proxyYAML, err := kubeadm.KubeProxyConfigurationToYAML(c.KubeProxyConfiguration)
	scriptConfig := InitScriptConfig{
		DockerConfigJSON: jsonutils.Marshal(c.DockerConfiguration).PrettyString(),
		CACert:           c.CACert,
		CAKey:            c.CAKey,
		EtcdCACert:       c.EtcdCACert,
		EtcdCAKey:        c.EtcdCAKey,
		FrontProxyCACert: c.FrontProxyCACert,
		FrontProxyCAKey:  c.FrontProxyCAKey,
		SaCert:           c.SaCert,
		SaKey:            c.SaKey,

		InitConfiguration:      initYAML,
		ClusterConfiguration:   clusterYAML,
		KubeProxyConfiguration: proxyYAML,
	}
	return scriptConfig.ToScript()
}

type JoinControlplaneConfig struct {
	DockerConfiguration *api.DockerConfig
	JoinConfiguration   *kubeadmv1beta1.JoinConfiguration

	CACert           string
	CAKey            string
	EtcdCACert       string
	EtcdCAKey        string
	FrontProxyCACert string
	FrontProxyCAKey  string
	SaCert           string
	SaKey            string
}

func (c JoinControlplaneConfig) ToScript() (string, error) {
	joinYAML, err := kubeadm.ConfigurationToYAML(c.JoinConfiguration)
	if err != nil {
		return "", err
	}
	scriptConfig := InitScriptConfig{
		DockerConfigJSON: jsonutils.Marshal(c.DockerConfiguration).PrettyString(),
		CACert:           c.CACert,
		CAKey:            c.CAKey,
		EtcdCACert:       c.EtcdCACert,
		EtcdCAKey:        c.EtcdCAKey,
		FrontProxyCACert: c.FrontProxyCACert,
		FrontProxyCAKey:  c.FrontProxyCAKey,
		SaCert:           c.SaCert,
		SaKey:            c.SaKey,

		ControlJoinConfiguration: joinYAML,
	}
	return scriptConfig.ToScript()
}

type JoinNodeConfig struct {
	DockerConfiguration *api.DockerConfig
	JoinConfiguration   *kubeadmv1beta1.JoinConfiguration
}

func (c JoinNodeConfig) ToScript() (string, error) {
	joinYAML, err := kubeadm.ConfigurationToYAML(c.JoinConfiguration)
	if err != nil {
		return "", err
	}
	scriptConfig := InitScriptConfig{
		DockerConfigJSON: jsonutils.Marshal(c.DockerConfiguration).PrettyString(),

		NodeJoinConfiguration: joinYAML,
	}
	return scriptConfig.ToScript()
}
