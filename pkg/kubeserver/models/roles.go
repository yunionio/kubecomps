package models

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/kubernetes/pkg/apis/rbac"
	"k8s.io/kubernetes/pkg/apis/rbac/validation"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
)

var (
	roleManager *SRoleManager
	_           IClusterModel = new(SRole)
)

func init() {
	GetRoleManager()
}

func GetRoleManager() *SRoleManager {
	if roleManager == nil {
		roleManager = NewK8sNamespaceModelManager(func() ISyncableManager {
			return &SRoleManager{
				SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
					SRole{},
					"roles_tbl",
					"rbacrole",
					"rbacroles",
					api.ResourceNameRole,
					rbacv1.GroupName,
					rbacv1.SchemeGroupVersion.Version,
					api.KindNameRole,
					new(rbacv1.Role),
				),
			}
		}).(*SRoleManager)
	}
	return roleManager
}

// +onecloud:swagger-gen-model-singular=rbacrole
// +onecloud:swagger-gen-model-plural=rbacroles
type SRoleManager struct {
	SNamespaceResourceBaseManager
}

type SRole struct {
	SNamespaceResourceBase
}

func (m *SRoleManager) GetRoleKind() string {
	return api.KindNameRole
}

func (m *SRoleManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.NamespaceResourceListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SNamespaceResourceBaseManager.ListItemFilter(ctx, q, userCred, input)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (m *SRoleManager) ValidateRoleObject(role *rbacv1.Role) error {
	return ValidateCreateK8sObject(role, new(rbac.Role), func(out interface{}) field.ErrorList {
		return validation.ValidateRole(out.(*rbac.Role))
	})
}

func (m *SRoleManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.RoleCreateInput) (*api.RoleCreateInput, error) {
	nInput, err := m.SNamespaceResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, &input.NamespaceResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.NamespaceResourceCreateInput = *nInput
	role, err := input.ToRole(input.Namespace)
	if err != nil {
		return nil, err
	}
	if err := m.ValidateRoleObject(role); err != nil {
		return nil, err
	}
	return input, nil
}

func (m *SRoleManager) NewRemoteObjectForCreate(obj IClusterModel, _ *client.ClusterManager, data jsonutils.JSONObject) (interface{}, error) {
	input := new(api.RoleCreateInput)
	if err := data.Unmarshal(input); err != nil {
		return nil, err
	}
	nsName, err := obj.(*SRole).GetNamespaceName()
	if err != nil {
		return nil, err
	}
	return input.ToRole(nsName)
}

func (obj *SRole) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.RoleUpdateInput) (*api.RoleUpdateInput, error) {
	if _, err := ValidateUpdateData(obj, ctx, userCred, input.JSON(input)); err != nil {
		return nil, err
	}
	return input, nil
}

func ValidateUpdateRoleObject(oldObj, newObj *rbacv1.Role) error {
	if err := ValidateUpdateK8sObject(oldObj, newObj, new(rbac.Role), new(rbac.Role), func(newObj, oldObj interface{}) field.ErrorList {
		return validation.ValidateRoleUpdate(oldObj.(*rbac.Role), newObj.(*rbac.Role))
	}); err != nil {
		return errors.Wrap(err, "ValidateUpdateRoleObject")
	}
	return nil
}

func (obj *SRole) NewRemoteObjectForUpdate(cli *client.ClusterManager, remoteObj interface{}, data jsonutils.JSONObject) (interface{}, error) {
	input := new(api.RoleUpdateInput)
	if err := data.Unmarshal(input); err != nil {
		return nil, errors.Wrap(err, "unmarshal json")
	}
	oldObj := remoteObj.(*rbacv1.Role)
	newObj := oldObj.DeepCopyObject().(*rbacv1.Role)
	newObj.Rules = input.Rules
	if err := ValidateUpdateRoleObject(oldObj, newObj); err != nil {
		return nil, err
	}
	return newObj, nil
}

func (m *SRoleManager) NewFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, obj interface{}) (IClusterModel, error) {
	model, err := m.SNamespaceResourceBaseManager.NewFromRemoteObject(ctx, userCred, cluster, obj)
	if err != nil {
		return nil, err
	}
	//kRole := obj.(*rbac.Role)
	//roleObj := model.(*SRole)
	return model, nil
}

func (r *SRole) UpdateFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, extObj interface{}) error {
	if err := r.SNamespaceResourceBase.UpdateFromRemoteObject(ctx, userCred, extObj); err != nil {
		return err
	}
	return nil
}

func (r *SRole) GetDetails(ctx context.Context, cli *client.ClusterManager, base interface{}, k8sObj runtime.Object, isList bool) interface{} {
	detail := r.SNamespaceResourceBase.GetDetails(ctx, cli, base, k8sObj, isList).(api.NamespaceResourceDetail)
	role := k8sObj.(*rbacv1.Role)
	out := api.RoleDetail{
		NamespaceResourceDetail: detail,
		Rules:                   role.Rules,
	}
	return out
}
