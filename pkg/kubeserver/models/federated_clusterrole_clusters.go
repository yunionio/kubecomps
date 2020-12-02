package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
)

var (
	FedClusterRoleClusterManager *SFedClusterRoleClusterManager
	_                            IFedJointClusterModel = new(SFedClusterRoleCluster)
)

func init() {
	db.InitManager(func() {
		FedClusterRoleClusterManager = NewFedJointManager(func() db.IJointModelManager {
			return &SFedClusterRoleClusterManager{
				SFedJointClusterManager: NewFedJointClusterManager(
					SFedClusterRoleCluster{},
					"federatedclusterroleclusters_tbl",
					"federatedclusterrolecluster",
					"federatedclusterroleclusters",
					GetFedClusterRoleManager(),
					GetClusterRoleManager(),
				),
			}
		}).(*SFedClusterRoleClusterManager)
		GetFedClusterRoleManager().SetJointModelManager(FedClusterRoleClusterManager)
		RegisterFedJointClusterManager(GetFedClusterRoleManager(), FedClusterRoleClusterManager)
	})
}

// +onecloud:swagger-gen-model-singular=federatedclusterrolecluster
// +onecloud:swagger-gen-model-plural=federatedclusterroleclusters
type SFedClusterRoleClusterManager struct {
	SFedJointClusterManager
}

type SFedClusterRoleCluster struct {
	SFedJointCluster
}

func (m *SFedClusterRoleClusterManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.FederatedClusterRoleClusterListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SFedJointClusterManager.ListItemFilter(ctx, q, userCred, &input.FedJointClusterListInput)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (obj *SFedClusterRoleCluster) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, obj)
}

func (obj *SFedClusterRoleCluster) GetFedClusterRole() (*SFedClusterRole, error) {
	fedObj, err := GetFedResAPI().JointResAPI().FetchFedResourceModel(obj)
	if err != nil {
		return nil, err
	}
	return fedObj.(*SFedClusterRole), nil
}

func (obj *SFedClusterRoleCluster) GetResourceCreateData(ctx context.Context, userCred mcclient.TokenCredential, fObj IFedModel, base api.NamespaceResourceCreateInput) (jsonutils.JSONObject, error) {
	fedObj := fObj.(*SFedClusterRole)
	input := api.ClusterRoleCreateInput{
		ClusterResourceCreateInput: base.ClusterResourceCreateInput,
		Rules:                      fedObj.Spec.Template.Rules,
	}
	return input.JSON(input), nil
}

func (obj *SFedClusterRoleCluster) GetResourceUpdateData(ctx context.Context, userCred mcclient.TokenCredential, fObj IFedModel, resObj IClusterModel, base api.NamespaceResourceUpdateInput) (jsonutils.JSONObject, error) {
	fedObj := fObj.(*SFedClusterRole)
	input := api.ClusterRoleUpdateInput{
		ClusterResourceUpdateInput: base.ClusterResourceUpdateInput,
		Rules:                      fedObj.Spec.Template.Rules,
	}
	return input.JSON(input), nil
}
