package api

const (
	ClusterCA        = "cluster-ca"
	EtcdCA           = "etcd-ca"
	FrontProxyCA     = "front-proxy-ca"
	ServiceAccountCA = "service-account"
)

// KeyPair is how operators can supply custom keypairs for kubeadm to use
type KeyPair struct {
	// base64 encoded cert and key
	Cert []byte `json:"cert"`
	Key  []byte `json:"key"`
}

// HasCertAndKey returns whether a keypair contains cert and key of non-zero length.
func (kp *KeyPair) HasCertAndKey() bool {
	return len(kp.Cert) != 0 && len(kp.Key) != 0
}

type X509KeyPairCreateInput struct {
	Name        string `json:"name"`
	User        string `json:"user"`
	ClusterId   string `json:"cluster_id"`
	Certificate string `json:"certificate"`
	PrivateKey  string `json:"private_key"`
}
