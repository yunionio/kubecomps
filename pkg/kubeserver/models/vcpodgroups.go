package models

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
)

var (
	vcPodGroupManager *SVCPodGroupManager
	_                 IClusterModel  = new(SVCPodGroup)
	_                 IPodOwnerModel = new(SVCPodGroup)
)

func init() {
	GetVCPodGroupManager()
}

type SVCPodGroupManager struct {
	SNamespaceResourceBaseManager
}

type SVCPodGroup struct {
	SNamespaceResourceBase

	Queue string `width:"256" charset:"utf8" nullable:"true" list:"user"`
}

func GetVCPodGroupManager() *SVCPodGroupManager {
	if vcPodGroupManager == nil {
		vcPodGroupManager = NewK8sNamespaceModelManager(func() ISyncableManager {
			return &SVCPodGroupManager{
				SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
					new(SVCPodGroup),
					"vcpodgroups_tbl",
					"vcpodgroup",
					"vcpodgroups",
					api.ResourceNameVolcanoPodGroup,
					"scheduling.volcano.sh",
					"v1beta1",
					api.KindNameVolcanoPodGroup,
					&unstructured.Unstructured{},
				),
			}
		}).(*SVCPodGroupManager)
	}
	return vcPodGroupManager
}

func (m *SVCPodGroupManager) ListRemoteObjects(cli *client.ClusterManager) ([]interface{}, error) {
	return listVolcanoRemoteObjects(cli, api.KindNameVolcanoPodGroup, m.SNamespaceResourceBaseManager.ListRemoteObjects)
}

func (m *SVCPodGroupManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.VCPodGroupListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SNamespaceResourceBaseManager.ListItemFilter(ctx, q, userCred, &input.NamespaceResourceListInput)
	if err != nil {
		return nil, err
	}
	if input.Queue != "" {
		q = q.Equals("queue", input.Queue)
	}
	return q, nil
}

func (m *SVCPodGroupManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.NamespaceResourceCreateInput) (*api.NamespaceResourceCreateInput, error) {
	return nil, httperrors.NewForbiddenError("podgroup is read-only")
}

func (obj *SVCPodGroup) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.NamespaceResourceUpdateInput) (*api.NamespaceResourceUpdateInput, error) {
	return nil, httperrors.NewForbiddenError("podgroup is read-only")
}

func (obj *SVCPodGroup) UpdateRawdata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, httperrors.NewForbiddenError("podgroup is read-only")
}

func (obj *SVCPodGroup) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	return httperrors.NewForbiddenError("podgroup is read-only")
}

func (obj *SVCPodGroup) GetDetails(ctx context.Context, cli *client.ClusterManager, base interface{}, k8sObj runtime.Object, isList bool) interface{} {
	detail := api.VCPodGroupDetail{
		NamespaceResourceDetail: obj.SNamespaceResourceBase.GetDetails(ctx, cli, base, k8sObj, isList).(api.NamespaceResourceDetail),
	}
	u, err := asUnstructured(k8sObj)
	if err != nil {
		detail.Queue = obj.Queue
		detail.Status = obj.Status
		return detail
	}
	detail.Queue = nestedString(u, "spec", "queue")
	detail.Job = volcanoJobOwnerName(u)
	detail.MinMember = nestedInt64(u, "spec", "minMember")
	detail.Status = volcanoPhase(u, "status", "phase")
	return detail
}

func (obj *SVCPodGroup) SetStatusByRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, extObj interface{}) error {
	if u, ok := extObj.(*unstructured.Unstructured); ok {
		obj.Status = volcanoPhase(u, "status", "phase")
		obj.Queue = nestedString(u, "spec", "queue")
		return nil
	}
	return obj.SNamespaceResourceBase.SetStatusByRemoteObject(ctx, userCred, extObj)
}

func (obj *SVCPodGroup) GetRawPods(cli *client.ClusterManager, rawObj runtime.Object) ([]*v1.Pod, error) {
	name := obj.GetName()
	namespace := ""
	if u, err := asUnstructured(rawObj); err == nil {
		if u.GetName() != "" {
			name = u.GetName()
		}
		namespace = u.GetNamespace()
	}
	if namespace == "" {
		ns, err := obj.GetNamespaceName()
		if err != nil {
			return []*v1.Pod{}, nil
		}
		namespace = ns
	}
	return volcanoPodsByPodGroup(cli, namespace, name)
}

func (obj *SVCPodGroup) GetDetailsRawdata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	raw, err := obj.SNamespaceResourceBase.GetDetailsRawdata(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return volcanoRawdata(raw, api.VolcanoPodGroupKind), nil
}
