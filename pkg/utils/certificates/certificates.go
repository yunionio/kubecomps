/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package certificates

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math"
	"math/big"
	"net"
	"strings"
	"time"

	kapi "k8s.io/client-go/tools/clientcmd/api"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
)

const (
	rsaKeySize     = 2048
	duration100y   = time.Hour * 24 * 365 * 100
	clusterCA      = api.ClusterCA
	etcdCA         = api.EtcdCA
	frontProxyCA   = api.FrontProxyCA
	serviceAccount = api.ServiceAccountCA
)

// NewPrivateKey creates an rSA private key
func NewPrivateKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, rsaKeySize)
}

// AltNames contains the domain names and IP addresses that will be added
// to the API Server's x509 certificate SubAltNames field. The values will
// be passed directly to the x509.Certificate object.
type AltNames struct {
	DNSNames []string
	IPs      []net.IP
}

// Config contains the basic fields required for creating a certificate
type Config struct {
	CommonName   string
	Organization []string
	AltNames     AltNames
	Usages       []x509.ExtKeyUsage
	Duration     time.Duration
}

func GetOrGenerateCACert(kp *api.KeyPair, user string) (api.KeyPair, error) {
	if kp == nil || !kp.HasCertAndKey() {
		log.V(2).Infof("Generating keypair for %q", user)
		x509Cert, privKey, err := NewCertificateAuthority()
		if err != nil {
			return api.KeyPair{}, errors.Wrapf(err, "failed to generate CA cert for %q", user)
		}
		if kp == nil {
			return api.KeyPair{
				Cert: EncodeCertPEM(x509Cert),
				Key:  EncodePrivateKeyPEM(privKey),
			}, nil
		}
		kp.Cert = EncodeCertPEM(x509Cert)
		kp.Key = EncodePrivateKeyPEM(privKey)
	}
	return *kp, nil
}

func GetOrGenerateServiceAccountKeys(kp *api.KeyPair, user string) (api.KeyPair, error) {
	if kp == nil || !kp.HasCertAndKey() {
		log.V(2).Infof("Generating service account keys for %q", user)
		saCreds, err := NewPrivateKey()
		if err != nil {
			return api.KeyPair{}, errors.Wrapf(err, "failed to create service account public and private keys")
		}
		saPub, err := EncodePublicKeyPEM(&saCreds.PublicKey)
		if err != nil {
			return api.KeyPair{}, errors.Wrapf(err, "failed to encode service account public key to PEM")
		}
		if kp == nil {
			return api.KeyPair{
				Cert: saPub,
				Key:  EncodePrivateKeyPEM(saCreds),
			}, nil
		}
		kp.Cert = saPub
		kp.Key = EncodePrivateKeyPEM(saCreds)
	}
	return *kp, nil
}

// NewSignedCert creates a signed certificate using the given CA certificate and key
func (cfg *Config) NewSignedCert(key *rsa.PrivateKey, caCert *x509.Certificate, caKey *rsa.PrivateKey) (*x509.Certificate, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate random integer for signed certificate")
	}

	if len(cfg.CommonName) == 0 {
		return nil, errors.Error("must specify a CommonName")
	}

	if len(cfg.Usages) == 0 {
		return nil, errors.Error("must specify at least one ExtKeyUsage")
	}

	if cfg.Duration == 0 {
		return nil, errors.Error("must specify duration")
	}

	tmpl := x509.Certificate{
		Subject: pkix.Name{
			CommonName:   cfg.CommonName,
			Organization: cfg.Organization,
		},
		DNSNames:     cfg.AltNames.DNSNames,
		IPAddresses:  cfg.AltNames.IPs,
		SerialNumber: serial,
		NotBefore:    caCert.NotBefore,
		NotAfter:     time.Now().Add(cfg.Duration).UTC(),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  cfg.Usages,
	}

	b, err := x509.CreateCertificate(rand.Reader, &tmpl, caCert, key.Public(), caKey)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create signed certificate: %+v", tmpl)
	}

	return x509.ParseCertificate(b)
}

// NewCertificateAuthority creates new certificate and private key for the certificate authority
func NewCertificateAuthority() (*x509.Certificate, *rsa.PrivateKey, error) {
	key, err := NewPrivateKey()
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to create private key")
	}

	cert, err := NewSelfSignedCACert(key)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to create self-signed certificate")
	}

	return cert, key, nil
}

// NewSelfSignedCACert creates a CA certificate.
func NewSelfSignedCACert(key *rsa.PrivateKey) (*x509.Certificate, error) {
	cfg := Config{
		CommonName: "kubernetes",
	}

	now := time.Now().UTC()

	tmpl := x509.Certificate{
		SerialNumber: new(big.Int).SetInt64(0),
		Subject: pkix.Name{
			CommonName:   cfg.CommonName,
			Organization: cfg.Organization,
		},
		NotBefore:             now,
		NotAfter:              now.Add(duration100y),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	b, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, key.Public(), key)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create self signed CA certificate: %+v", tmpl)
	}

	return x509.ParseCertificate(b)
}

// NewKubeconfig creates a new Kubeconfig where endpoint is the ELB endpoint.
func NewKubeconfig(clusterName, endpoint string, caCert *x509.Certificate, caKey *rsa.PrivateKey) (*kapi.Config, error) {
	cfg := &Config{
		CommonName:   "kubernetes-admin",
		Organization: []string{"system:masters"},
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		Duration:     duration100y,
	}

	return NewKubeconfigV2(clusterName, endpoint, caCert, caKey, cfg)
}

// EncodeCertPEM returns PEM-endcoded certificate data.
func EncodeCertPEM(cert *x509.Certificate) []byte {
	block := pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}
	return pem.EncodeToMemory(&block)
}

// EncodePrivateKeyPEM returns PEM-encoded private key data.
func EncodePrivateKeyPEM(key *rsa.PrivateKey) []byte {
	block := pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	return pem.EncodeToMemory(&block)
}

// EncodePublicKeyPEM returns PEM-encoded public key data.
func EncodePublicKeyPEM(key *rsa.PublicKey) ([]byte, error) {
	der, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return []byte{}, err
	}
	block := pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: der,
	}
	return pem.EncodeToMemory(&block), nil
}

// DecodeCertPEM attempts to return a decoded certificate or nil
// if the encoded input does not contain a certificate.
func DecodeCertPEM(encoded []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(encoded)
	if block == nil {
		return nil, nil
	}

	return x509.ParseCertificate(block.Bytes)
}

// DecodePrivateKeyPEM attempts to return a decoded key or nil
// if the encoded input does not contain a private key.
func DecodePrivateKeyPEM(encoded []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(encoded)
	if block == nil {
		return nil, nil
	}

	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

// GenerateCertificateHash returns the encoded sha256 hash for the certificate provided
func GenerateCertificateHash(encoded []byte) (string, error) {
	cert, err := DecodeCertPEM(encoded)
	if err != nil || cert == nil {
		return "", errors.Errorf("failed to parse PEM block containing the public key")
	}

	certHash := sha256.Sum256(cert.RawSubjectPublicKeyInfo)
	return "sha256:" + strings.ToLower(hex.EncodeToString(certHash[:])), nil
}

func NewKubeconfigV2(clusterName, endpoint string, caCert *x509.Certificate, caKey *rsa.PrivateKey, cfg *Config) (*kapi.Config, error) {
	if cfg == nil {
		return nil, errors.Errorf("config must provided")
	}

	clientKey, err := NewPrivateKey()
	if err != nil {
		return nil, errors.Wrap(err, "unable to create private key")
	}

	clientCert, err := cfg.NewSignedCert(clientKey, caCert, caKey)
	if err != nil {
		return nil, errors.Wrap(err, "unable to sign certificate")
	}

	userName := cfg.CommonName
	contextName := fmt.Sprintf("%s@%s", userName, clusterName)
	return &kapi.Config{
		Clusters: map[string]*kapi.Cluster{
			clusterName: {
				Server:                   endpoint,
				CertificateAuthorityData: EncodeCertPEM(caCert),
			},
		},
		Contexts: map[string]*kapi.Context{
			contextName: {
				Cluster:  clusterName,
				AuthInfo: userName,
			},
		},
		AuthInfos: map[string]*kapi.AuthInfo{
			userName: {
				ClientKeyData:         EncodePrivateKeyPEM(clientKey),
				ClientCertificateData: EncodeCertPEM(clientCert),
			},
		},
		CurrentContext: contextName,
	}, nil
}
