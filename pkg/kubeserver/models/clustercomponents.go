package models

import (
	"context"
	"database/sql"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SClusterComponentManager struct {
	SClusterJointsManager
}

var ClusterComponentManager *SClusterComponentManager

func init() {
	db.InitManager(func() {
		ClusterComponentManager = &SClusterComponentManager{
			SClusterJointsManager: NewClusterJointsManager(
				SClusterComponent{},
				"clustercomponents_tbl",
				"kubeclustercomponent",
				"kubeclustercomponents",
				ComponentManager),
		}
		ClusterComponentManager.SetVirtualObject(ClusterComponentManager)
		ClusterComponentManager.TableSpec().AddIndex(true, "component_id", "cluster_id")
	})
}

// +onecloud:swagger-gen-ignore
type SClusterComponent struct {
	SClusterJointsBase

	ComponentId string `width:"36" charset:"ascii" create:"required" list:"user"`
}

func (m *SClusterComponentManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (m *SClusterComponentManager) GetComponentsByCluster(clusterId string) ([]SComponent, error) {
	q := ComponentManager.Query()
	subq := m.Query("component_id").Equals("cluster_id", clusterId).SubQuery()
	q.In("id", subq)
	cs := make([]SComponent, 0)
	err := db.FetchModelObjects(ComponentManager, q, &cs)
	return cs, err
}

func (m *SClusterComponentManager) GetByComponent(componentId string) ([]SClusterComponent, error) {
	q := m.Query().Equals("component_id", componentId)
	ret := make([]SClusterComponent, 0)
	if err := db.FetchModelObjects(m, q, &ret); err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return ret, nil
}

func (joint *SClusterComponent) Master() db.IStandaloneModel {
	return db.JointMaster(joint)
}

func (joint *SClusterComponent) Slave() db.IStandaloneModel {
	return db.JointSlave(joint)
}

func (joint *SClusterComponentManager) GetMasterFieldName() string {
	return "cluster_id"
}

func (joint *SClusterComponentManager) GetSlaveFieldName() string {
	return "component_id"
}

func (joint *SClusterComponent) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, joint)
}

func (joint *SClusterComponent) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, joint)
}

func (joint *SClusterComponent) GetComponent() (*SComponent, error) {
	obj, err := ComponentManager.FetchById(joint.ComponentId)
	if err != nil {
		return nil, err
	}
	return obj.(*SComponent), nil
}

func (joint *SClusterComponent) DoSave(ctx context.Context) error {
	if err := ClusterComponentManager.TableSpec().Insert(ctx, joint); err != nil {
		return err
	}
	joint.SetModelManager(ClusterComponentManager, joint)
	return nil
}
