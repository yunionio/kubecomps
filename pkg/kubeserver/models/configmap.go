package models

import (
	"context"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
)

var (
	configMapManager *SConfigMapManager
	_                IPodOwnerModel = new(SConfigMap)
)

func init() {
	GetConfigMapManager()
}

func GetConfigMapManager() *SConfigMapManager {
	if configMapManager == nil {
		configMapManager = NewK8sNamespaceModelManager(func() ISyncableManager {
			return &SConfigMapManager{
				SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
					SConfigMap{},
					"configmaps_tbl",
					"configmap",
					"configmaps",
					api.ResourceNameConfigMap,
					v1.GroupName,
					v1.SchemeGroupVersion.Version,
					api.KindNameConfigMap,
					new(v1.ConfigMap),
				),
			}
		}).(*SConfigMapManager)
	}
	return configMapManager
}

// +onecloud:swagger-gen-model-singular=configmap
// +onecloud:swagger-gen-model-plural=configmaps
type SConfigMapManager struct {
	SNamespaceResourceBaseManager
}

type SConfigMap struct {
	SNamespaceResourceBase
}

func (m SConfigMapManager) GetRawConfigMaps(cluster *client.ClusterManager, ns string) ([]*v1.ConfigMap, error) {
	indexer := cluster.GetHandler().GetIndexer()
	return indexer.ConfigMapLister().ConfigMaps(ns).List(labels.Everything())
}

func (m *SConfigMapManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.ConfigMapCreateInput) (*api.ConfigMapCreateInput, error) {
	if _, err := m.SNamespaceResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, &input.NamespaceResourceCreateInput); err != nil {
		return input, err
	}
	if len(input.Data) == 0 {
		return nil, httperrors.NewNotAcceptableError("data is empty")
	}
	return input, nil
}

func (m *SConfigMapManager) NewRemoteObjectForCreate(model IClusterModel, cli *client.ClusterManager, body jsonutils.JSONObject) (interface{}, error) {
	input := new(api.ConfigMapCreateInput)
	if err := body.Unmarshal(input); err != nil {
		return nil, errors.Wrap(err, "unmarshal to configmap input")
	}
	objMeta, err := input.ToObjectMeta(model.(api.INamespaceGetter))
	if err != nil {
		return nil, err
	}
	return &v1.ConfigMap{
		ObjectMeta: objMeta,
		Data:       input.Data,
	}, nil
}

func (m *SConfigMap) NewRemoteObjectForUpdate(cli *client.ClusterManager, remoteObj interface{}, body jsonutils.JSONObject) (interface{}, error) {
	input := new(api.ConfigMapUpdateInput)
	if err := body.Unmarshal(input); err != nil {
		return nil, err
	}
	k8sObj := remoteObj.(*v1.ConfigMap)
	for k, v := range input.Data {
		k8sObj.Data[k] = v
	}
	return k8sObj, nil
}

func (obj *SConfigMap) GetRawPods(cli *client.ClusterManager, rawObj runtime.Object) ([]*v1.Pod, error) {
	cfgName := obj.GetName()
	rawPods, err := GetPodManager().GetRawPodsByObjectNamespace(cli, rawObj)
	if err != nil {
		return nil, err
	}
	mountPods := make([]*v1.Pod, 0)
	markMap := make(map[string]bool, 0)
	for _, pod := range rawPods {
		cfgs := GetPodConfigMapVolumes(pod)
		for _, cfg := range cfgs {
			if cfg.ConfigMap.Name == cfgName {
				if _, ok := markMap[pod.GetName()]; !ok {
					mountPods = append(mountPods, pod)
					markMap[pod.GetName()] = true
				}
			}
		}
	}
	return mountPods, err
}
