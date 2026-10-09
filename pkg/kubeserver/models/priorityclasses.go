package models

import (
	"context"

	schedulingv1 "k8s.io/api/scheduling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
)

var (
	priorityClassManager *SPriorityClassManager
	_                    IClusterModel = new(SPriorityClass)
)

func init() {
	GetPriorityClassManager()
}

func GetPriorityClassManager() *SPriorityClassManager {
	if priorityClassManager == nil {
		priorityClassManager = NewK8sModelManager(func() ISyncableManager {
			return &SPriorityClassManager{
				SClusterResourceBaseManager: NewClusterResourceBaseManager(
					new(SPriorityClass),
					"priorityclasses_tbl",
					"priorityclass",
					"priorityclasses",
					api.ResourceNamePriorityClass,
					schedulingv1.GroupName,
					schedulingv1.SchemeGroupVersion.Version,
					api.KindNamePriorityClass,
					new(schedulingv1.PriorityClass),
				),
			}
		}).(*SPriorityClassManager)
	}
	return priorityClassManager
}

type SPriorityClassManager struct {
	SClusterResourceBaseManager
}

type SPriorityClass struct {
	SClusterResourceBase
}

func (m *SPriorityClassManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.PriorityClassCreateInput) (*api.PriorityClassCreateInput, error) {
	cInput, err := m.SClusterResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, &input.ClusterResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.ClusterResourceCreateInput = *cInput
	if input.Name == "" {
		return nil, httperrors.NewNotEmptyError("name is empty")
	}
	if isSystemPriorityClass(input.Name) {
		return nil, httperrors.NewForbiddenError("priority class %s is reserved", input.Name)
	}
	return input, nil
}

func (m *SPriorityClassManager) NewRemoteObjectForCreate(_ IClusterModel, _ *client.ClusterManager, data jsonutils.JSONObject) (interface{}, error) {
	input := new(api.PriorityClassCreateInput)
	if err := data.Unmarshal(input); err != nil {
		return nil, errors.Wrap(err, "unmarshal priority class create input")
	}
	obj := &schedulingv1.PriorityClass{
		TypeMeta: metav1.TypeMeta{
			APIVersion: schedulingv1.SchemeGroupVersion.String(),
			Kind:       api.KindNamePriorityClass,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: input.Name,
		},
		Value:         input.Value,
		GlobalDefault: input.GlobalDefault,
		Description:   input.Description,
	}
	return obj, nil
}

func (obj *SPriorityClass) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if isSystemPriorityClass(obj.Name) {
		return httperrors.NewForbiddenError("priority class %s cannot be deleted", obj.Name)
	}
	return obj.SClusterResourceBase.ValidateDeleteCondition(ctx, info)
}

func (obj *SPriorityClass) GetDetails(ctx context.Context, cli *client.ClusterManager, base interface{}, k8sObj runtime.Object, isList bool) interface{} {
	detail := obj.SClusterResourceBase.GetDetails(ctx, cli, base, k8sObj, isList).(api.ClusterResourceDetail)
	pc, ok := k8sObj.(*schedulingv1.PriorityClass)
	if !ok || pc == nil {
		return api.PriorityClassDetail{ClusterResourceDetail: detail}
	}
	return api.PriorityClassDetail{
		ClusterResourceDetail: detail,
		Value:                 pc.Value,
		GlobalDefault:         pc.GlobalDefault,
		Description:           pc.Description,
	}
}

func isSystemPriorityClass(name string) bool {
	return name == "system-cluster-critical" || name == "system-node-critical"
}
