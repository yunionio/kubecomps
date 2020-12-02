package models

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	ocapis "yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/utils/certificates"
)

var X509KeyPairManager *SX509KeyPairManager

func init() {
	X509KeyPairManager = &SX509KeyPairManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SX509KeyPair{},
			"x509keypairs_tbl",
			"x509keypair",
			"x509keypairs",
		),
	}
	X509KeyPairManager.SetVirtualObject(X509KeyPairManager)
}

type SX509KeyPairManager struct {
	db.SVirtualResourceBaseManager
}

// +onecloud:swagger-gen-ignore
type SX509KeyPair struct {
	db.SVirtualResourceBase

	User        string `width:"256" charset:"ascii" nullable:"false" get:"user" create:"required"`
	Certificate string `nullable:"false" create:"required"`
	PrivateKey  string `nullable:"false" create:"required"`
}

func (m *SX509KeyPairManager) generateName(cluster *SCluster, ownerId mcclient.IIdentityProvider, user string) (string, error) {
	hint := fmt.Sprintf("%s-%s", cluster.GetName(), user)
	return db.GenerateName(X509KeyPairManager, ownerId, hint)
}

func (m *SX509KeyPairManager) createKeyPair(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, kp api.KeyPair, user string) (*SX509KeyPair, error) {
	name, err := m.generateName(cluster, userCred, user)
	if err != nil {
		return nil, err
	}
	input := &api.X509KeyPairCreateInput{
		Name:        name,
		User:        user,
		Certificate: string(kp.Cert),
		PrivateKey:  string(kp.Key),
	}
	data := jsonutils.Marshal(input)
	obj, err := db.DoCreate(m, ctx, userCred, nil, data, userCred)
	if err != nil {
		return nil, errors.Wrapf(err, "Create x509keypair object for cluster: %s", cluster.GetName())
	}
	kpObj := obj.(*SX509KeyPair)
	if err := cluster.AttachKeypair(ctx, userCred, kpObj); err != nil {
		return nil, errors.Wrap(err, "cluster attach keypair")
	}
	return kpObj, nil
}

func (m *SX509KeyPairManager) GenerateCertificates(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, user string) (*SX509KeyPair, error) {
	kp, err := certificates.GetOrGenerateCACert(nil, user)
	if err != nil {
		return nil, errors.Wrap(err, "generate CA cert")
	}
	kpObj, err := m.createKeyPair(ctx, userCred, cluster, kp, user)
	if err != nil {
		return nil, errors.Wrapf(err, "create %s keypair", user)
	}
	return kpObj, nil
}

func (m *SX509KeyPairManager) GenerateServiceAccountKeys(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, user string) (*SX509KeyPair, error) {
	kp, err := certificates.GetOrGenerateServiceAccountKeys(nil, user)
	if err != nil {
		return nil, errors.Wrap(err, "generate service account keys")
	}
	return m.createKeyPair(ctx, userCred, cluster, kp, user)
}

func (m *SX509KeyPairManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	input := new(api.X509KeyPairCreateInput)
	if err := data.Unmarshal(input); err != nil {
		return nil, httperrors.NewInputParameterError("Unmarshal create input: %v", err)
	}
	cert := []byte(input.Certificate)
	if input.User != api.ServiceAccountCA {
		if _, err := certificates.DecodeCertPEM(cert); err != nil {
			return nil, httperrors.NewInputParameterError("Invalid Certificate: %v", err)
		}
	}
	key := []byte(input.PrivateKey)
	if _, err := certificates.DecodePrivateKeyPEM(key); err != nil {
		return nil, httperrors.NewInputParameterError("Invalid Certificate PrivateKey: %v", err)
	}
	ocInput := ocapis.VirtualResourceCreateInput{}
	data.Unmarshal(&ocInput)
	data = jsonutils.Marshal(input).(*jsonutils.JSONDict)
	if _, err := m.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, ocInput); err != nil {
		return nil, err
	}
	return data, nil
}

func (m *SX509KeyPairManager) DeleteKeyPairsByCluster(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster) error {
	kps, err := ClusterX509KeyPairManager.GetKeyPairsByCluster(cluster.GetId())
	if err != nil {
		return errors.Wrap(err, "GetKeyPairsByClusters")
	}
	for _, kp := range kps {
		if err := kp.Delete(ctx, userCred); err != nil {
			return errors.Wrapf(err, "Delete keypair %s", kp.GetName())
		}
	}
	return nil
}

func (kp *SX509KeyPair) GetClusterKeypair() (*SClusterX509KeyPair, error) {
	clusterKeypair, err := db.NewModelObject(ClusterX509KeyPairManager)
	if err != nil {
		return nil, errors.Wrap(err, "new cluster keypair model")
	}
	q := ClusterX509KeyPairManager.Query().Equals("keypair_id", kp.Id)
	if err := q.First(clusterKeypair); err != nil {
		return nil, errors.Errorf("Get cluster joint keypair error: %v", err)
	}
	return clusterKeypair.(*SClusterX509KeyPair), nil
}

func (kp *SX509KeyPair) GetCluster() (*SCluster, error) {
	ckp, err := kp.GetClusterKeypair()
	if err != nil {
		return nil, err
	}
	return ckp.GetCluster(), nil
}

func (kp *SX509KeyPair) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	ckp, err := kp.GetClusterKeypair()
	if err != nil {
		return errors.Wrapf(err, "delete cluster joint  keypair")
	}
	if ckp != nil {
		if err := ckp.Detach(ctx, userCred); err != nil {
			return errors.Wrapf(err, "detach keypair %s joint", kp.GetName())
		}
	}
	return kp.SVirtualResourceBase.Delete(ctx, userCred)
}

func (kp *SX509KeyPair) HasCertAndKey() bool {
	return len(kp.Certificate) != 0 && len(kp.PrivateKey) != 0
}

func (kp *SX509KeyPair) ToKeyPair() *api.KeyPair {
	return &api.KeyPair{
		Cert: []byte(kp.Certificate),
		Key:  []byte(kp.PrivateKey),
	}
}

func (kp *SX509KeyPair) getExtraInfo(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	cluster, _ := kp.GetCluster()
	if cluster != nil {
		extra.Add(jsonutils.NewString(cluster.GetName()), "cluster")
		extra.Add(jsonutils.NewString(cluster.GetId()), "cluster_id")
	}
	return extra
}
