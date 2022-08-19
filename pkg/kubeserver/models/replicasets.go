package models

import (
	"context"
	extensions "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/version"
	"strconv"
	"yunion.io/x/kubecomps/pkg/kubeserver/k8s/common/model"

	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
)

var (
	replicaSetManager *SReplicaSetManager
	_                 IClusterModel = new(SReplicaSet)
)

func init() {
	GetReplicaSetManager()
}

// +onecloud:swagger-gen-model-singular=replicaset
// +onecloud:swagger-gen-model-plural=replicasets
type SReplicaSetManager struct {
	SNamespaceResourceBaseManager
}

type SReplicaSet struct {
	SNamespaceResourceBase
}

func GetReplicaSetManager() *SReplicaSetManager {
	if replicaSetManager == nil {
		replicaSetManager = NewK8sNamespaceModelManager(func() ISyncableManager {
			return &SReplicaSetManager{
				SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
					SReplicaSet{},
					"replicasets_tbl",
					"replicaset",
					"replicasets",
					api.ResourceNameReplicaSet,
					v1.GroupName,
					v1.SchemeGroupVersion.Version,
					api.KindNameReplicaSet,
					new(apps.ReplicaSet),
				),
			}
		}).(*SReplicaSetManager)
	}
	return replicaSetManager
}

func (m *SReplicaSetManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewBadRequestError("Not support replicasets create")
}

func (m *SReplicaSetManager) GetK8sResourceInfo(serverVersion *version.Info) model.K8sResourceInfo {
	if serverVersion != nil {
		if minor, err := strconv.Atoi(serverVersion.Minor); err == nil && minor >= 16 {
			return model.K8sResourceInfo{
				ResourceName: api.ResourceNameReplicaSet,
				Group:        apps.GroupName,
				Version:      v1.SchemeGroupVersion.Version,
				KindName:     api.KindNameReplicaSet,
				Object:       new(apps.ReplicaSet),
			}
		}
	}

	return model.K8sResourceInfo{
		ResourceName: api.ResourceNameReplicaSet,
		Group:        extensions.GroupName,
		Version:      "v1beta1",
		KindName:     api.KindNameReplicaSet,
		Object:       new(extensions.ReplicaSet),
	}
}

func (m *SReplicaSetManager) GetRawReplicaSets(cluster *client.ClusterManager, namespace string, selector labels.Selector) ([]*apps.ReplicaSet, error) {
	indexer := cluster.GetHandler().GetIndexer()
	list, err := indexer.ReplicaSetLister().ByNamespace(namespace).List(selector)
	if err != nil {
		return nil, err
	}
	res := make([]*apps.ReplicaSet, len(list))
	for idx, l := range list {
		newObj := &apps.ReplicaSet{}
		if err = runtime.DefaultUnstructuredConverter.FromUnstructured(l.(*unstructured.Unstructured).Object, newObj); err != nil {
			return nil, err
		}
		res[idx] = newObj
	}
	return res, nil
}

func (obj *SReplicaSet) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (api.ReplicaSetDetail, error) {
	return api.ReplicaSetDetail{}, nil
}

func (obj *SReplicaSet) GetRawPods(cli *client.ClusterManager, rawObj runtime.Object) ([]*v1.Pod, error) {
	return GetRawPodsByController(cli, rawObj.(metav1.Object))
}

func (obj *SReplicaSet) GetPodInfo(cli *client.ClusterManager, rs *apps.ReplicaSet) (*api.PodInfo, error) {
	pods, err := obj.GetRawPods(cli, rs)
	if err != nil {
		return nil, errors.Wrap(err, "replicaset get raw pods")
	}
	return GetPodInfo(rs.Status.Replicas, rs.Spec.Replicas, pods)
}

func (obj *SReplicaSet) GetDetails(cli *client.ClusterManager, base interface{}, k8sObj runtime.Object, isList bool) interface{} {
	detail := api.ReplicaSetDetail{
		NamespaceResourceDetail: obj.SNamespaceResourceBase.GetDetails(cli, base, k8sObj, isList).(api.NamespaceResourceDetail),
	}
	rs := k8sObj.(*apps.ReplicaSet)
	podInfo, err := obj.GetPodInfo(cli, rs)
	if err != nil {
		log.Errorf("get pod info error: %v", err)
		return detail
	}
	detail.Pods = *podInfo
	detail.InitContainerImages = GetInitContainerImages(&rs.Spec.Template.Spec)
	detail.ContainerImages = GetContainerImages(&rs.Spec.Template.Spec)
	return detail
}
