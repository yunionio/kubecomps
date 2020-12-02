package userdata

import (
	"encoding/base64"
	"fmt"
	"strings"

	"yunion.io/x/pkg/errors"
)

const (
	controlPlaneCloudInit = `{{.Header}}
write_files:
-   path: /etc/docker/daemon.json
    encoding: "base64"
    owner: root:root
    permissions: '0644'
    content: |
      {{.DockerConfig | Base64Encode}}

-   path: /etc/kubernetes/pki/ca.crt
    encoding: "base64"
    owner: root:root
    permissions: '0640'
    content: |
      {{.CACert | Base64Encode}}

-   path: /etc/kubernetes/pki/ca.key
    encoding: "base64"
    owner: root:root
    permissions: '0600'
    content: |
      {{.CAKey | Base64Encode}}

-   path: /etc/kubernetes/pki/etcd/ca.crt
    encoding: "base64"
    owner: root:root
    permissions: '0640'
    content: |
      {{.EtcdCACert | Base64Encode}}

-   path: /etc/kubernetes/pki/etcd/ca.key
    encoding: "base64"
    owner: root:root
    permissions: '0600'
    content: |
      {{.EtcdCAKey | Base64Encode}}

-   path: /etc/kubernetes/pki/front-proxy-ca.crt
    encoding: "base64"
    owner: root:root
    permissions: '0640'
    content: |
      {{.FrontProxyCACert | Base64Encode}}

-   path: /etc/kubernetes/pki/front-proxy-ca.key
    encoding: "base64"
    owner: root:root
    permissions: '0600'
    content: |
      {{.FrontProxyCAKey | Base64Encode}}

-   path: /etc/kubernetes/pki/sa.pub
    encoding: "base64"
    owner: root:root
    permissions: '0640'
    content: |
      {{.SaCert | Base64Encode}}

-   path: /etc/kubernetes/pki/sa.key
    encoding: "base64"
    owner: root:root
    permissions: '0600'
    content: |
      {{.SaKey | Base64Encode}}

-   path: /run/kubeadm.yaml
    owner: root:root
    permissions: '0640'
    content: |
      ---
{{.ClusterConfiguration | Indent 6}}
      ---
{{.InitConfiguration | Indent 6}}
      ---
{{.KubeProxyConfiguration | Indent 6}}
`

	controlPlaneJoinCloudInit = `{{.Header}}
write_files:
-   path: /etc/docker/daemon.json
    owner: root:root
    permissions: '0644'
    encoding: "base64"
    content: |
      {{.DockerConfig | Base64Encode}}

-   path: /etc/kubernetes/pki/ca.crt
    encoding: "base64"
    owner: root:root
    permissions: '0640'
    content: |
      {{.CACert | Base64Encode}}

-   path: /etc/kubernetes/pki/ca.key
    encoding: "base64"
    owner: root:root
    permissions: '0600'
    content: |
      {{.CAKey | Base64Encode}}

-   path: /etc/kubernetes/pki/etcd/ca.crt
    encoding: "base64"
    owner: root:root
    permissions: '0640'
    content: |
      {{.EtcdCACert | Base64Encode}}

-   path: /etc/kubernetes/pki/etcd/ca.key
    encoding: "base64"
    owner: root:root
    permissions: '0600'
    content: |
      {{.EtcdCAKey | Base64Encode}}

-   path: /etc/kubernetes/pki/front-proxy-ca.crt
    encoding: "base64"
    owner: root:root
    permissions: '0640'
    content: |
      {{.FrontProxyCACert | Base64Encode}}

-   path: /etc/kubernetes/pki/front-proxy-ca.key
    encoding: "base64"
    owner: root:root
    permissions: '0600'
    content: |
      {{.FrontProxyCAKey | Base64Encode}}

-   path: /etc/kubernetes/pki/sa.pub
    encoding: "base64"
    owner: root:root
    permissions: '0640'
    content: |
      {{.SaCert | Base64Encode}}

-   path: /etc/kubernetes/pki/sa.key
    encoding: "base64"
    owner: root:root
    permissions: '0600'
    content: |
      {{.SaKey | Base64Encode}}

-   path: /run/kubeadm-controlplane-join-config.yaml
    owner: root:root
    permissions: '0640'
    content: |
{{.JoinConfiguration | Indent 6}}
kubeadm:
  operation: join
  config: /run/kubeadm-controlplane-join-config.yaml
`
)

func isKeyPairValid(cert, key string) bool {
	return cert != "" && key != ""
}

// ControlPlaneInputCloudInit defines the context to generate a controlplane instance user data
type ControlPlaneInputCloudInit struct {
	baseUserDataCloudInit

	DockerConfig           string
	CACert                 string
	CAKey                  string
	EtcdCACert             string
	EtcdCAKey              string
	FrontProxyCACert       string
	FrontProxyCAKey        string
	SaCert                 string
	SaKey                  string
	ClusterConfiguration   string
	InitConfiguration      string
	KubeProxyConfiguration string
}

// ControlPlaneJoinInputCloudInit defines context to generate controlplane instance user data for controlplane node join
type ControlPlaneJoinInputCloudInit struct {
	baseUserDataCloudInit

	DockerConfig      string
	CACert            string
	CAKey             string
	EtcdCACert        string
	EtcdCAKey         string
	FrontProxyCACert  string
	FrontProxyCAKey   string
	SaCert            string
	SaKey             string
	BootstrapToken    string
	ELBAddress        string
	JoinConfiguration string
}

func (cpi *ControlPlaneInputCloudInit) validateCertificates() error {
	if !isKeyPairValid(cpi.CACert, cpi.CAKey) {
		return fmt.Errorf("CA cert material in the ControlPlaneInput is missing cert/key")
	}
	if !isKeyPairValid(cpi.EtcdCACert, cpi.EtcdCAKey) {
		return fmt.Errorf("ETCD CA cert material in the ControlPlaneInput is missing cert/key")
	}
	if !isKeyPairValid(cpi.FrontProxyCACert, cpi.FrontProxyCAKey) {
		return fmt.Errorf("FrontProxy CA cert material in ControlPlaneInput is missing cert/key")
	}
	if !isKeyPairValid(cpi.SaCert, cpi.SaKey) {
		return fmt.Errorf("ServiceAccount cert material in ControlPlaneInput is missing cert/key")
	}
	return nil
}

func (cpi *ControlPlaneJoinInputCloudInit) validateCertificates() error {
	if !isKeyPairValid(cpi.CACert, cpi.CAKey) {
		return fmt.Errorf("CA cert material in the ControlPlaneInput is missing cert/key")
	}
	if !isKeyPairValid(cpi.EtcdCACert, cpi.EtcdCAKey) {
		return fmt.Errorf("ETCD CA cert material in the ControlPlaneInput is missing cert/key")
	}
	if !isKeyPairValid(cpi.FrontProxyCACert, cpi.FrontProxyCAKey) {
		return fmt.Errorf("FrontProxy CA cert material in ControlPlaneInput is missing cert/key")
	}
	if !isKeyPairValid(cpi.SaCert, cpi.SaKey) {
		return fmt.Errorf("ServiceAccount cert material in ControlPlaneInput is missing cert/key")
	}
	return nil
}

// NewControlPlaneCloudInit returns the user data string to be used on a controlplane instance
func NewControlPlaneCloudInit(input *ControlPlaneInputCloudInit) (string, error) {
	input.Header = cloudConfigHeader
	if err := input.validateCertificates(); err != nil {
		return "", errors.Wrapf(err, "ControlPlaneInput is invalid")
	}

	fMap := map[string]interface{}{
		"Base64Encode": templateBase64Encode,
		"Indent":       templateYAMLIndent,
	}

	userData, err := generateWithFuncs("controlplane", controlPlaneCloudInit, funcMap(fMap), input)
	if err != nil {
		return "", errors.Wrapf(err, "failed to generate user data for new control plane machine")
	}

	return userData, err
}

// NewJoinControlPlaneCloudInit returns the user data string to be used on a new controlplane instance
func NewJoinControlPlaneCloudInit(input *ControlPlaneJoinInputCloudInit) (string, error) {
	input.Header = cloudConfigHeader

	if err := input.validateCertificates(); err != nil {
		return "", errors.Wrapf(err, "ControlPlaneInput is invalid")
	}

	fMap := map[string]interface{}{
		"Base64Encode": templateBase64Encode,
		"Indent":       templateYAMLIndent,
	}

	userData, err := generateWithFuncs("controlplane", controlPlaneJoinCloudInit, funcMap(fMap), input)
	if err != nil {
		return "", errors.Wrapf(err, "failed to generate user data for machine joining control plane")
	}
	return userData, err
}

func templateBase64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func templateYAMLIndent(i int, input string) string {
	split := strings.Split(input, "\n")
	ident := "\n" + strings.Repeat(" ", i)
	return strings.Repeat(" ", i) + strings.Join(split, ident)
}
