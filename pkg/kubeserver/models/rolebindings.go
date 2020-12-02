package models

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/kubernetes/pkg/apis/rbac"
	"k8s.io/kubernetes/pkg/apis/rbac/validation"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
)

var (
	roleBindingManager *SRoleBindingManager
	_                  db.IModel = new(SRoleBinding)
)

func init() {
	GetRoleBindingManager()
}

func GetRoleBindingManager() *SRoleBindingManager {
	if roleBindingManager == nil {
		roleBindingManager = NewK8sNamespaceModelManager(func() ISyncableManager {
			return &SRoleBindingManager{
				SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
					SRoleBinding{},
					"rolebindings_tbl",
					"rbacrolebinding",
					"rbacrolebindings",
					api.ResourceNameRoleBinding,
					rbacv1.GroupName,
					rbacv1.SchemeGroupVersion.Version,
					api.KindNameRoleBinding,
					new(rbac.RoleBinding),
				),
			}
		}).(*SRoleBindingManager)
	}
	return roleBindingManager
}

// +onecloud:swagger-gen-model-singular=rbacrolebinding
// +onecloud:swagger-gen-model-plural=rbacrolebindings
type SRoleBindingManager struct {
	SNamespaceResourceBaseManager
	SRoleRefResourceBaseManager
}

type SRoleBinding struct {
	SNamespaceResourceBase
	// SRoleRefResourceBase
}

func (m *SRoleBindingManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.NamespaceResourceListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SNamespaceResourceBaseManager.ListItemFilter(ctx, q, userCred, input)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (m *SRoleBindingManager) ValidateRoleBinding(rb *rbacv1.RoleBinding) error {
	return ValidateCreateK8sObject(rb, new(rbac.RoleBinding), func(out interface{}) field.ErrorList {
		return validation.ValidateRoleBinding(out.(*rbac.RoleBinding))
	})
}

func (m *SRoleBindingManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.RoleBindingCreateInput) (*api.RoleBindingCreateInput, error) {
	nInput, err := m.SNamespaceResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, &input.NamespaceResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.NamespaceResourceCreateInput = *nInput

	// var roleMan IRoleBaseManager
	if input.RoleRef.Kind == api.KindNameRole {
		// roleMan = GetRoleManager()
	} else if input.RoleRef.Kind == api.KindNameClusterRole {
		// roleMan = GetClusterRoleManager()
	} else {
		return nil, httperrors.NewNotAcceptableError("not support role ref kind %s", input.RoleRef.Kind)
	}
	/*
	 * if err := m.SRoleRefResourceBaseManager.ValidateRoleRef(roleMan, userCred, &input.RoleRef); err != nil {
	 *     return nil, err
	 * }
	 */

	rb, err := input.ToRoleBinding(input.Namespace)
	if err := m.ValidateRoleBinding(rb); err != nil {
		return nil, err
	}
	return input, nil
}

func (obj *SRoleBinding) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if err := obj.SNamespaceResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data); err != nil {
		return err
	}
	input := new(api.RoleBindingCreateInput)
	if err := data.Unmarshal(input); err != nil {
		return errors.Wrap(err, "unmarshal rolebinding create input")
	}
	/*
	 * if err := obj.SRoleRefResourceBase.CustomizeCreate(&input.RoleRef); err != nil {
	 *     return err
	 * }
	 */
	return nil
}

func (m *SRoleBindingManager) NewFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, obj interface{}) (IClusterModel, error) {
	model, err := m.SNamespaceResourceBaseManager.NewFromRemoteObject(ctx, userCred, cluster, obj)
	if err != nil {
		return nil, err
	}
	return model, nil
}

func (rb *SRoleBinding) UpdateFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, extObj interface{}) error {
	if err := rb.SNamespaceResourceBase.UpdateFromRemoteObject(ctx, userCred, extObj); err != nil {
		return err
	}
	return nil
}

func (rb *SRoleBinding) GetDetails(cli *client.ClusterManager, base interface{}, k8sObj runtime.Object, isList bool) interface{} {
	detail := rb.SNamespaceResourceBase.GetDetails(cli, base, k8sObj, isList).(api.NamespaceResourceDetail)
	binding := k8sObj.(*rbacv1.RoleBinding)
	out := api.RoleBindingDetail{
		NamespaceResourceDetail: detail,
		RoleRef:                 binding.RoleRef,
		Subjects:                binding.Subjects,
	}
	return out
}

func (m *SRoleBindingManager) NewRemoteObjectForCreate(obj IClusterModel, _ *client.ClusterManager, data jsonutils.JSONObject) (interface{}, error) {
	input := new(api.RoleBindingCreateInput)
	if err := data.Unmarshal(input); err != nil {
		return nil, err
	}
	nsName, err := obj.(*SRoleBinding).GetNamespaceName()
	if err != nil {
		return nil, err
	}
	return input.ToRoleBinding(nsName)
}

func ValidateUpdateRoleBindingObject(oldObj, newObj *rbacv1.RoleBinding) error {
	if err := ValidateUpdateK8sObject(oldObj, newObj, new(rbac.RoleBinding), new(rbac.RoleBinding), func(newObj, oldObj interface{}) field.ErrorList {
		return validation.ValidateRoleBindingUpdate(oldObj.(*rbac.RoleBinding), newObj.(*rbac.RoleBinding))
	}); err != nil {
		return errors.Wrap(err, "ValidateUpdateRoleBindingObject")
	}
	return nil
}

func (rb *SRoleBinding) NewRemoteObjectForUpdate(cli *client.ClusterManager, remoteObj interface{}, data jsonutils.JSONObject) (interface{}, error) {
	input := new(api.RoleBindingUpdateInput)
	if err := data.Unmarshal(input); err != nil {
		return nil, err
	}
	oldObj := remoteObj.(*rbacv1.RoleBinding)
	newObj := oldObj.DeepCopyObject().(*rbacv1.RoleBinding)
	newObj.Subjects = input.Subjects
	newObj.RoleRef = rbacv1.RoleRef(input.RoleRef)
	if err := ValidateUpdateRoleBindingObject(oldObj, newObj); err != nil {
		return nil, err
	}
	return newObj, nil
}
