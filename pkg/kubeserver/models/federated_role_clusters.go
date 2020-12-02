package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
)

var (
	FedRoleClusterManager *SFedRoleClusterManager
	_                     IFedJointClusterModel = new(SFedRoleCluster)
)

func init() {
	db.InitManager(func() {
		FedRoleClusterManager = NewFedJointManager(func() db.IJointModelManager {
			return &SFedRoleClusterManager{
				SFedNamespaceJointClusterManager: NewFedNamespaceJointClusterManager(
					SFedRoleCluster{},
					"federatedroleclusters_tbl",
					"federatedrolecluster",
					"federatedroleclusters",
					GetFedRoleManager(),
					GetRoleManager(),
				),
			}
		}).(*SFedRoleClusterManager)
		GetFedRoleManager().SetJointModelManager(FedRoleClusterManager)
		RegisterFedJointClusterManager(GetFedRoleManager(), FedRoleClusterManager)
	})
}

// +onecloud:swagger-gen-model-singular=federatedrolecluster
// +onecloud:swagger-gen-model-plural=federatedroleclusters
type SFedRoleClusterManager struct {
	SFedNamespaceJointClusterManager
}

type SFedRoleCluster struct {
	SFederatedNamespaceJointCluster
}

func (obj *SFedRoleCluster) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, obj)
}

func (obj *SFedRoleCluster) GetResourceCreateData(ctx context.Context, userCred mcclient.TokenCredential, fObj IFedModel, base api.NamespaceResourceCreateInput) (jsonutils.JSONObject, error) {
	fedObj := fObj.(*SFedRole)
	input := api.RoleCreateInput{
		NamespaceResourceCreateInput: base,
		Rules:                        fedObj.Spec.Template.Rules,
	}
	return input.JSON(input), nil
}

func (obj *SFedRoleCluster) GetResourceUpdateData(ctx context.Context, userCred mcclient.TokenCredential, fObj IFedModel, resObj IClusterModel, base api.NamespaceResourceUpdateInput) (jsonutils.JSONObject, error) {
	fedObj := fObj.(*SFedRole)
	input := api.RoleUpdateInput{
		NamespaceResourceUpdateInput: base,
		Rules:                        fedObj.Spec.Template.Rules,
	}
	return input.JSON(input), nil
}
