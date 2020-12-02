package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
)

var (
	FedRoleBindingClusterManager *SFedRoleBindingClusterManager
	_                            IFedJointClusterModel = new(SFedRoleBindingCluster)
)

func init() {
	db.InitManager(func() {
		FedRoleBindingClusterManager = NewFedJointManager(func() db.IJointModelManager {
			return &SFedRoleBindingClusterManager{
				SFedNamespaceJointClusterManager: NewFedNamespaceJointClusterManager(
					SFedRoleBindingCluster{},
					"federatedrolebindingclusters_tbl",
					"federatedrolebindingcluster",
					"federatedrolebindingclusters",
					GetFedRoleBindingManager(),
					GetRoleBindingManager(),
				),
			}
		}).(*SFedRoleBindingClusterManager)
		GetFedRoleBindingManager().SetJointModelManager(FedRoleBindingClusterManager)
		RegisterFedJointClusterManager(GetFedRoleBindingManager(), FedRoleBindingClusterManager)
	})
}

// +onecloud:swagger-gen-model-singular=federatedrolebindingcluster
// +onecloud:swagger-gen-model-plural=federatedrolebindingclusters
type SFedRoleBindingClusterManager struct {
	SFedNamespaceJointClusterManager
}

type SFedRoleBindingCluster struct {
	SFederatedNamespaceJointCluster
}

func (obj *SFedRoleBindingCluster) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, obj)
}

func (obj *SFedRoleBindingCluster) GetResourceCreateData(ctx context.Context, userCred mcclient.TokenCredential, fObj IFedModel, base api.NamespaceResourceCreateInput) (jsonutils.JSONObject, error) {
	fedObj := fObj.(*SFedRoleBinding)
	input := api.RoleBindingCreateInput{
		NamespaceResourceCreateInput: base,
		Subjects:                     fedObj.Spec.Template.Subjects,
		RoleRef:                      api.RoleRef(fedObj.Spec.Template.RoleRef),
	}
	return input.JSON(input), nil
}

func (obj *SFedRoleBindingCluster) GetResourceUpdateData(ctx context.Context, userCred mcclient.TokenCredential, fObj IFedModel, resObj IClusterModel, base api.NamespaceResourceUpdateInput) (jsonutils.JSONObject, error) {
	fedObj := fObj.(*SFedRoleBinding)
	input := api.RoleBindingUpdateInput{
		NamespaceResourceUpdateInput: base,
		Subjects:                     fedObj.Spec.Template.Subjects,
		RoleRef:                      api.RoleRef(fedObj.Spec.Template.RoleRef),
	}
	return input.JSON(input), nil
}
