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
	FedClusterRoleBindingClusterManager *SFedClusterRoleBindingClusterManager
)

func init() {
	db.InitManager(func() {
		FedClusterRoleBindingClusterManager = NewFedJointManager(func() db.IJointModelManager {
			return &SFedClusterRoleBindingClusterManager{
				SFedJointClusterManager: NewFedJointClusterManager(
					SFedClusterRoleBindingCluster{},
					"federatedclusterrolebindingclusters_tbl",
					"federatedclusterrolebindingcluster",
					"federatedclusterrolebindingclusters",
					GetFedClusterRoleBindingManager(),
					GetClusterRoleBindingManager(),
				),
			}
		}).(*SFedClusterRoleBindingClusterManager)
		GetFedClusterRoleBindingManager().SetJointModelManager(FedClusterRoleBindingClusterManager)
		RegisterFedJointClusterManager(GetFedClusterRoleBindingManager(), FedClusterRoleBindingClusterManager)
	})
}

// +onecloud:swagger-gen-model-singular=federatedclusterrolebindingcluster
// +onecloud:swagger-gen-model-plural=federatedclusterrolebindingclusters
type SFedClusterRoleBindingClusterManager struct {
	SFedJointClusterManager
}

type SFedClusterRoleBindingCluster struct {
	SFedJointCluster
}

func (m *SFedClusterRoleBindingClusterManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.FedClusterRoleBindingClusterListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SFedJointClusterManager.ListItemFilter(ctx, q, userCred, &input.FedJointClusterListInput)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (obj *SFedClusterRoleBindingCluster) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, obj)
}

func (obj *SFedClusterRoleBindingCluster) GetResourceCreateData(ctx context.Context, userCred mcclient.TokenCredential, fObj IFedModel, base api.NamespaceResourceCreateInput) (jsonutils.JSONObject, error) {
	fedObj := fObj.(*SFedClusterRoleBinding)
	input := api.ClusterRoleBindingCreateInput{
		ClusterResourceCreateInput: base.ClusterResourceCreateInput,
		Subjects:                   fedObj.Spec.Template.Subjects,
		RoleRef:                    api.RoleRef(fedObj.Spec.Template.RoleRef),
	}
	return input.JSON(input), nil
}

func (obj *SFedClusterRoleBindingCluster) GetResourceUpdateData(ctx context.Context, userCred mcclient.TokenCredential, fObj IFedModel, resObj IClusterModel, base api.NamespaceResourceUpdateInput) (jsonutils.JSONObject, error) {
	fedObj := fObj.(*SFedClusterRoleBinding)
	input := api.ClusterRoleBindingUpdateInput{
		ClusterResourceUpdateInput: base.ClusterResourceUpdateInput,
		Subjects:                   fedObj.Spec.Template.Subjects,
		RoleRef:                    api.RoleRef(fedObj.Spec.Template.RoleRef),
	}
	return input.JSON(input), nil
}
