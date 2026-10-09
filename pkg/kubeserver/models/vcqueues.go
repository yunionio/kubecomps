package models

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
)

var (
	vcQueueManager *SVCQueueManager
	_              IClusterModel  = new(SVCQueue)
	_              IPodOwnerModel = new(SVCQueue)
)

func init() {
	GetVCQueueManager()
}

type SVCQueueManager struct {
	SClusterResourceBaseManager
}

type SVCQueue struct {
	SClusterResourceBase
}

func GetVCQueueManager() *SVCQueueManager {
	if vcQueueManager == nil {
		vcQueueManager = NewK8sModelManager(func() ISyncableManager {
			return &SVCQueueManager{
				SClusterResourceBaseManager: NewClusterResourceBaseManager(
					new(SVCQueue),
					"vcqueues_tbl",
					"vcqueue",
					"vcqueues",
					api.ResourceNameVolcanoQueue,
					"scheduling.volcano.sh",
					"v1beta1",
					api.KindNameVolcanoQueue,
					&unstructured.Unstructured{},
				),
			}
		}).(*SVCQueueManager)
	}
	return vcQueueManager
}

func (m *SVCQueueManager) ListRemoteObjects(cli *client.ClusterManager) ([]interface{}, error) {
	return listVolcanoRemoteObjects(cli, api.KindNameVolcanoQueue, m.SClusterResourceBaseManager.ListRemoteObjects)
}

func (m *SVCQueueManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.VCQueueCreateInput) (*api.VCQueueCreateInput, error) {
	cInput, err := m.SClusterResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, &input.ClusterResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.ClusterResourceCreateInput = *cInput
	if input.Name == "" {
		return nil, httperrors.NewNotEmptyError("name is empty")
	}
	if isSystemQueue(input.Name) {
		return nil, httperrors.NewForbiddenError("queue %s is reserved", input.Name)
	}
	if err := validateQueueSpec(&input.VCQueueSpecInput); err != nil {
		return nil, err
	}
	return input, nil
}

func (m *SVCQueueManager) NewRemoteObjectForCreate(_ IClusterModel, _ *client.ClusterManager, data jsonutils.JSONObject) (interface{}, error) {
	input := new(api.VCQueueCreateInput)
	if err := data.Unmarshal(input); err != nil {
		return nil, errors.Wrap(err, "unmarshal queue create input")
	}
	return newQueueObject(input.Name, &input.VCQueueSpecInput)
}

func (obj *SVCQueue) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.VCQueueUpdateInput) (*api.VCQueueUpdateInput, error) {
	base, err := obj.SClusterResourceBase.ValidateUpdateData(ctx, userCred, query, &input.ClusterResourceUpdateInput)
	if err != nil {
		return nil, err
	}
	input.ClusterResourceUpdateInput = *base
	if err := validateQueueSpec(&input.VCQueueSpecInput); err != nil {
		return nil, err
	}
	return input, nil
}

func (obj *SVCQueue) NewRemoteObjectForUpdate(_ *client.ClusterManager, remoteObj interface{}, data jsonutils.JSONObject) (interface{}, error) {
	input := new(api.VCQueueUpdateInput)
	if err := data.Unmarshal(input); err != nil {
		return nil, errors.Wrap(err, "unmarshal queue update input")
	}
	u, err := asUnstructured(remoteObj.(runtime.Object))
	if err != nil {
		return nil, err
	}
	if err := applyQueueSpec(u, &input.VCQueueSpecInput); err != nil {
		return nil, err
	}
	return u, nil
}

func (obj *SVCQueue) UpdateRemoteObject(remoteObj interface{}) (interface{}, error) {
	u, err := asUnstructured(remoteObj.(runtime.Object))
	if err != nil {
		return nil, err
	}
	cli, err := obj.GetClusterClient()
	if err != nil {
		return nil, err
	}
	return replaceVolcanoUnstructured(cli, api.ResourceNameVolcanoQueue, u)
}

func (obj *SVCQueue) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if isSystemQueue(obj.Name) {
		return httperrors.NewForbiddenError("queue %s cannot be deleted", obj.Name)
	}
	return obj.SClusterResourceBase.ValidateDeleteCondition(ctx, info)
}

func (obj *SVCQueue) GetDetails(ctx context.Context, cli *client.ClusterManager, base interface{}, k8sObj runtime.Object, isList bool) interface{} {
	detailBase := obj.SClusterResourceBase.GetDetails(ctx, cli, base, k8sObj, isList).(api.ClusterResourceDetail)
	u, err := asUnstructured(k8sObj)
	if err != nil {
		out := api.VCQueueDetail{ClusterResourceDetail: detailBase, Status: obj.Status}
		return out
	}
	return queueDetailFromUnstructured(detailBase, u)
}

func (obj *SVCQueue) SetStatusByRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, extObj interface{}) error {
	if u, ok := extObj.(*unstructured.Unstructured); ok {
		obj.Status = volcanoPhase(u, "status", "state")
		return nil
	}
	return obj.SClusterResourceBase.SetStatusByRemoteObject(ctx, userCred, extObj)
}

func (obj *SVCQueue) GetRawPods(cli *client.ClusterManager, rawObj runtime.Object) ([]*v1.Pod, error) {
	name := obj.GetName()
	if u, err := asUnstructured(rawObj); err == nil && u.GetName() != "" {
		name = u.GetName()
	}
	return volcanoPodsByQueue(cli, name)
}

func (obj *SVCQueue) GetDetailsRawdata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	raw, err := obj.SClusterResourceBase.GetDetailsRawdata(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return volcanoRawdata(raw, api.VolcanoQueueKind), nil
}
