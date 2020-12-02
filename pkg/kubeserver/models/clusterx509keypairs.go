package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SClusterX509KeyPairManager struct {
	SClusterJointsManager
}

var ClusterX509KeyPairManager *SClusterX509KeyPairManager

func init() {
	db.InitManager(func() {
		ClusterX509KeyPairManager = &SClusterX509KeyPairManager{
			SClusterJointsManager: NewClusterJointsManager(
				SClusterX509KeyPair{},
				"clusterx509keypairs_tbl",
				"clusterx509keypair",
				"clusterx509keypairs",
				X509KeyPairManager,
			),
		}
		ClusterX509KeyPairManager.SetVirtualObject(ClusterX509KeyPairManager)
		ClusterX509KeyPairManager.TableSpec().AddIndex(true, "keypair_id", "cluster_id")
	})
}

// +onecloud:swagger-gen-ignore
type SClusterX509KeyPair struct {
	SClusterJointsBase

	KeypairId string `width:"36" charset:"ascii" create:"required" list:"user"`
	User      string `width:"256" charset:"ascii" nullable:"false" get:"user" create:"required"`
}

func (man *SClusterX509KeyPairManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (m *SClusterX509KeyPairManager) GetKeyPairsByClusters(clusterIds []string) ([]SX509KeyPair, error) {
	q := X509KeyPairManager.Query()
	subq := m.Query("keypair_id").In("cluster_id", clusterIds).SubQuery()
	q.In("id", subq)
	kps := make([]SX509KeyPair, 0)
	err := db.FetchModelObjects(X509KeyPairManager, q, &kps)
	return kps, err
}

func (m *SClusterX509KeyPairManager) GetKeyPairsByCluster(clusterId string) ([]SX509KeyPair, error) {
	return m.GetKeyPairsByClusters([]string{clusterId})
}

func (m *SClusterX509KeyPairManager) GetKeyPairByClusterUser(clusterId string, user string) (*SX509KeyPair, error) {
	clusterKp := SClusterX509KeyPair{}
	if err := m.Query().Equals("cluster_id", clusterId).Equals("user", user).First(&clusterKp); err != nil {
		return nil, errors.Wrapf(err, "Get cluster %s %s keypair", clusterId, user)
	}
	return clusterKp.GetKeypair()
}

func (man *SClusterX509KeyPair) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return false
}

func (joint *SClusterX509KeyPairManager) GetMasterFieldName() string {
	return "cluster_id"
}

func (joint *SClusterX509KeyPairManager) GetSlaveFieldName() string {
	return "keypair_id"
}

func (joint *SClusterX509KeyPair) Master() db.IStandaloneModel {
	return db.JointMaster(joint)
}

func (joint *SClusterX509KeyPair) Slave() db.IStandaloneModel {
	return db.JointSlave(joint)
}

func (joint *SClusterX509KeyPair) GetKeypair() (*SX509KeyPair, error) {
	kp, err := X509KeyPairManager.FetchById(joint.KeypairId)
	if err != nil {
		return nil, errors.Errorf("Get x509 keypair by id %s: %v", joint.KeypairId, err)
	}
	return kp.(*SX509KeyPair), nil
}

func (joint *SClusterX509KeyPair) getExtraInfo(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	keypair, _ := joint.GetKeypair()
	if keypair == nil {
		return extra
	}
	extra.Add(jsonutils.NewString(keypair.GetName()), "keypair")
	extra.Add(jsonutils.NewString(keypair.User), "user")
	return extra
}

func (joint *SClusterX509KeyPair) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, joint)
}

func (joint *SClusterX509KeyPair) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, joint)
}
