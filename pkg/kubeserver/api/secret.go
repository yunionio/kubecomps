package api

import (
	"k8s.io/api/core/v1"
)

const (
	SecretTypeCephCSI v1.SecretType = "yunion.io/ceph-csi"
)

type SecretCreateInput struct {
	NamespaceResourceCreateInput
	Type             v1.SecretType                      `json:"type"`
	DockerConfigJson *DockerConfigJsonSecretCreateInput `json:"dockerConfigJson"`
	CephCSI          *CephCSISecretCreateInput          `json:"cephCSI"`
}

type DockerConfigJsonSecretCreateInput struct {
	// required: true
	User string `json:"user"`
	// required: true
	Password string `json:"password"`
	// required: true
	Server string `json:"server"`
	Email  string `json:"email"`
}

type RegistrySecretCreateInput struct {
	K8sNamespaceResourceCreateInput
	DockerConfigJsonSecretCreateInput
}

type CephCSISecretCreateInput struct {
	// required: true
	UserId string `json:"userId"`
	// required: true
	UserKey string `json:"userKey"`

	EncryptionPassphrase string `json:"encryptionPassphrase"`
}

// Secret is a single secret returned to the frontend.
type Secret struct {
	ObjectMeta
	TypeMeta
	Type v1.SecretType `json:"type"`
}

// SecretDetail API resource provides mechanisms to inject containers with configuration data while keeping
// containers agnostic of Kubernetes
type SecretDetail struct {
	Secret

	// Data contains the secret data.  Each key must be a valid DNS_SUBDOMAIN
	// or leading dot followed by valid DNS_SUBDOMAIN.
	// The serialized form of the secret data is a base64 encoded string,
	// representing the arbitrary (possibly non-string) data value here.
	Data map[string][]byte `json:"data"`
}

type SecretDetailV2 struct {
	NamespaceResourceDetail
	Type v1.SecretType `json:"type"`
	// Data contains the secret data.  Each key must be a valid DNS_SUBDOMAIN
	// or leading dot followed by valid DNS_SUBDOMAIN.
	// The serialized form of the secret data is a base64 encoded string,
	// representing the arbitrary (possibly non-string) data value here.
	Data map[string][]byte `json:"data"`
}

type SecretListInput struct {
	NamespaceResourceListInput
	Type string `json:"type"`
}
