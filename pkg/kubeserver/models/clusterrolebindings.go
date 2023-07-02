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
	clusterRoleBindingManager *SClusterRoleBindingManager
	_                         IClusterModel = new(SClusterRoleBinding)
)

func init() {
	GetClusterRoleBindingManager()
}

func GetClusterRoleBindingManager() *SClusterRoleBindingManager {
	if clusterRoleBindingManager == nil {
		clusterRoleBindingManager = NewK8sModelManager(func() ISyncableManager {
			return &SClusterRoleBindingManager{
				SClusterResourceBaseManager: NewClusterResourceBaseManager(
					SClusterRoleBinding{},
					"clusterrolebindings_tbl",
					"rbacclusterrolebinding",
					"rbacclusterrolebindings",
					api.ResourceNameClusterRoleBinding,
					rbacv1.GroupName,
					rbacv1.SchemeGroupVersion.Version,
					api.KindNameClusterRoleBinding,
					new(rbac.ClusterRoleBinding),
				),
			}
		}).(*SClusterRoleBindingManager)
	}
	return clusterRoleBindingManager
}

// +onecloud:swagger-gen-model-singular=rbacclusterrolebinding
// +onecloud:swagger-gen-model-plural=rbacclusterrolebindings
type SClusterRoleBindingManager struct {
	SClusterResourceBaseManager
	// SRoleRefResourceBaseManager
}

type SClusterRoleBinding struct {
	SClusterResourceBase
	// SRoleRefResourceBase
}

func (m *SClusterRoleBindingManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.ClusterResourceListInput) (*sqlchemy.SQuery, error) {
	return m.SClusterResourceBaseManager.ListItemFilter(ctx, q, userCred, input)
}

func (m *SClusterRoleBindingManager) SyncResources(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster) error {
	return SyncClusterResources(ctx, userCred, cluster, m)
}

func (m *SClusterRoleBindingManager) ValidateClusterRoleBinding(crb *rbacv1.ClusterRoleBinding) error {
	return ValidateCreateK8sObject(crb, new(rbac.ClusterRoleBinding), func(out interface{}) field.ErrorList {
		return validation.ValidateClusterRoleBinding(out.(*rbac.ClusterRoleBinding))
	})
}

func (m *SClusterRoleBindingManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.ClusterRoleBindingCreateInput) (*api.ClusterRoleBindingCreateInput, error) {
	cInput, err := m.SClusterResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, &input.ClusterResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.ClusterResourceCreateInput = *cInput

	/*
	 * if err := m.SRoleRefResourceBaseManager.ValidateRoleRef(GetClusterRoleManager(), userCred, &input.RoleRef); err != nil {
	 *     return nil, err
	 * }
	 */

	crb := input.ToClusterRoleBinding()
	if err := m.ValidateClusterRoleBinding(crb); err != nil {
		return nil, err
	}
	return input, nil
}

func (obj *SClusterRoleBinding) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if err := obj.SClusterResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data); err != nil {
		return err
	}
	input := new(api.ClusterRoleBindingCreateInput)
	if err := data.Unmarshal(input); err != nil {
		return errors.Wrap(err, "unmarshal clusterrolebinding create input")
	}
	/*
	 * if err := obj.SRoleRefResourceBase.CustomizeCreate(&input.RoleRef); err != nil {
	 *     return err
	 * }
	 */
	return nil
}

func (m *SClusterRoleBindingManager) NewFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, obj interface{}) (IClusterModel, error) {
	return m.SClusterResourceBaseManager.NewFromRemoteObject(ctx, userCred, cluster, obj)
}

func (m *SClusterRoleBindingManager) NewRemoteObjectForCreate(obj IClusterModel, _ *client.ClusterManager, data jsonutils.JSONObject) (interface{}, error) {
	input := new(api.ClusterRoleBindingCreateInput)
	if err := data.Unmarshal(input); err != nil {
		return nil, err
	}
	return input.ToClusterRoleBinding(), nil
}

func ValidateUpdateClusterRoleBindingObject(oldObj, newObj *rbacv1.ClusterRoleBinding) error {
	if err := ValidateUpdateK8sObject(oldObj, newObj, new(rbac.ClusterRoleBinding), new(rbac.ClusterRoleBinding), func(newObj, oldObj interface{}) field.ErrorList {
		return validation.ValidateClusterRoleBindingUpdate(oldObj.(*rbac.ClusterRoleBinding), newObj.(*rbac.ClusterRoleBinding))
	}); err != nil {
		return errors.Wrap(err, "ValidateUpdateClusterRoleBindingObject")
	}
	return nil
}

func (crb *SClusterRoleBinding) NewRemoteObjectForUpdate(cli *client.ClusterManager, remoteObj interface{}, data jsonutils.JSONObject) (interface{}, error) {
	input := new(api.ClusterRoleBindingUpdateInput)
	if err := data.Unmarshal(input); err != nil {
		return nil, err
	}
	oldObj := remoteObj.(*rbacv1.ClusterRoleBinding)
	newObj := oldObj.DeepCopyObject().(*rbacv1.ClusterRoleBinding)
	newObj.Subjects = input.Subjects
	newObj.RoleRef = rbacv1.RoleRef(input.RoleRef)
	if err := ValidateUpdateClusterRoleBindingObject(oldObj, newObj); err != nil {
		return nil, err
	}
	return newObj, nil
}

func (crb *SClusterRoleBinding) UpdateFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, extObj interface{}) error {
	return crb.SClusterResourceBase.UpdateFromRemoteObject(ctx, userCred, extObj)
}

func (crb *SClusterRoleBinding) GetDetails(ctx context.Context, cli *client.ClusterManager, base interface{}, k8sObj runtime.Object, isList bool) interface{} {
	detail := crb.SClusterResourceBase.GetDetails(ctx, cli, base, k8sObj, isList).(api.ClusterResourceDetail)
	binding := k8sObj.(*rbacv1.ClusterRoleBinding)
	out := api.ClusterRoleBindingDetail{
		ClusterResourceDetail: detail,
		RoleRef:               binding.RoleRef,
		Subjects:              binding.Subjects,
	}
	return out
}
