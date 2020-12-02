package models

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/kubernetes/pkg/apis/rbac"
	"k8s.io/kubernetes/pkg/apis/rbac/validation"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
)

var (
	fedRoleManager *SFedRoleManager
	_              IFedModelManager = new(SFedRoleManager)
	_              IFedModel        = new(SFedRole)
)

func init() {
	GetFedRoleManager()
}

func GetFedRoleManager() *SFedRoleManager {
	if fedRoleManager == nil {
		fedRoleManager = newModelManager(func() db.IModelManager {
			return &SFedRoleManager{
				SFedNamespaceResourceManager: NewFedNamespaceResourceManager(
					SFedRole{},
					"federatedroles_tbl",
					"federatedrole",
					"federatedroles",
				),
			}
		}).(*SFedRoleManager)
	}
	return fedRoleManager
}

// +onecloud:swagger-gen-model-singular=federatedrole
// +onecloud:swagger-gen-model-plural=federatedroles
type SFedRoleManager struct {
	SFedNamespaceResourceManager
}

type SFedRole struct {
	SFedNamespaceResource
	Spec *api.FederatedRoleSpec `list:"user" update:"user" create:"required"`
}

func (m *SFedRoleManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.FedRoleCreateInput) (*api.FedRoleCreateInput, error) {
	nInput, err := m.SFedNamespaceResourceManager.ValidateCreateData(ctx, userCred, ownerCred, query, &input.FederatedNamespaceResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.FederatedNamespaceResourceCreateInput = *nInput
	if err := GetRoleManager().ValidateRoleObject(input.ToRole(nInput.Federatednamespace)); err != nil {
		return nil, err
	}
	return input, nil
}

func (m *SFedRoleManager) GetPropertyApiResources(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (api.ClusterAPIGroupResources, error) {
	ret, err := GetFedClustersApiResources(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return []api.ClusterAPIGroupResource{}, nil
	}
	return ret.(api.ClusterAPIGroupResources), nil
}

func (m *SFedRoleManager) GetPropertyClusterUsers(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (api.ClusterUsers, error) {
	ret, err := GetFedClustersUsers(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return []api.ClusterUser{}, nil
	}
	return ret.(api.ClusterUsers), nil
}

func (m *SFedRoleManager) GetPropertyClusterUserGroups(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (api.ClusterUserGroups, error) {
	ret, err := GetFedClustersUserGroups(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return []api.ClusterUserGroup{}, nil
	}
	return ret.(api.ClusterUserGroups), nil
}

func ValidateUpdateFedRoleObject(oldObj, newObj *rbacv1.Role) error {
	if err := ValidateUpdateK8sObject(oldObj, newObj, new(rbac.Role), new(rbac.Role), func(newObj, oldObj interface{}) field.ErrorList {
		return validation.ValidateRole(newObj.(*rbac.Role))
	}); err != nil {
		return errors.Wrap(err, "ValidateUpdateRoleObject")
	}
	return nil
}

func (obj *SFedRole) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.FedRoleUpdateInput) (*api.FedRoleUpdateInput, error) {
	bInput, err := obj.SFedNamespaceResource.ValidateUpdateData(ctx, userCred, query, &input.FedNamespaceResourceUpdateInput)
	if err != nil {
		return nil, err
	}
	input.FedNamespaceResourceUpdateInput = *bInput
	objMeta := obj.GetK8sObjectMeta()
	oldObj := obj.Spec.ToRole(objMeta)
	newObj := input.ToRole(objMeta)
	if err := ValidateUpdateFedRoleObject(oldObj, newObj); err != nil {
		return nil, err
	}
	return input, nil
}
