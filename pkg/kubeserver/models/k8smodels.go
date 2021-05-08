package models

import (
	"context"
	"reflect"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/object"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
	"yunion.io/x/kubecomps/pkg/kubeserver/k8s/common/model"
	"yunion.io/x/kubecomps/pkg/kubeserver/models/manager"
)

var globalK8sModelManagers map[model.K8sResourceInfo]IK8sModelManager

func init() {
	model.GetK8sModelManagerByKind = func(kindName string) model.IModelManager {
		return GetK8sResourceManagerByKind(kindName).(model.IModelManager)
	}
}

type IK8sModelManager interface {
	lockman.ILockedClass
	object.IObject

	GetK8sResourceInfo() model.K8sResourceInfo
}

func RegisterK8sModelManager(man IK8sModelManager) {
	if globalK8sModelManagers == nil {
		globalK8sModelManagers = make(map[model.K8sResourceInfo]IK8sModelManager)
	}
	globalK8sModelManagers[man.GetK8sResourceInfo()] = man
}

func GetOriginK8sModelManager(kindName string) IK8sModelManager {
	for rsInfo, man := range globalK8sModelManagers {
		if rsInfo.KindName == kindName {
			return man
		}
	}
	return nil
}

// GetK8sResourceManagerByKind used by bidirect sync
func GetK8sResourceManagerByKind(kindName string) manager.IK8sResourceManager {
	man := GetOriginK8sModelManager(kindName)
	if man == nil {
		return nil
	}
	return man.(manager.IK8sResourceManager)
}

func GetK8sModelManagerByKind(kindName string) model.IK8sModelManager {
	man := GetOriginK8sModelManager(kindName)
	if man == nil {
		return nil
	}
	return man.(model.IK8sModelManager)
}

func newModelManager(factory func() db.IModelManager) db.IModelManager {
	man := factory()
	man.SetVirtualObject(man)
	return man
}

func newK8sModelManager(factoryF func() ISyncableManager) ISyncableManager {
	man := newModelManager(func() db.IModelManager {
		return factoryF()
	}).(ISyncableManager)
	man.InitOwnedManager(man)
	RegisterK8sModelManager(man)
	return man
}

func NewK8sModelManager(factoryF func() ISyncableManager) ISyncableManager {
	man := newK8sModelManager(factoryF)
	GetClusterManager().AddSubManager(man)
	return man
}

func NewK8sNamespaceModelManager(factoryF func() ISyncableManager) ISyncableManager {
	man := newK8sModelManager(factoryF)
	GetNamespaceManager().AddSubManager(man)
	return man
}

type SK8sOwnedResourceBaseManager struct {
	ownedManager IClusterModelManager
}

func newK8sOwnedResourceManager(ownedMan IClusterModelManager) SK8sOwnedResourceBaseManager {
	return SK8sOwnedResourceBaseManager{ownedManager: ownedMan}
}

type IK8sOwnedResource interface {
	IsOwnedBy(ownerModel IClusterModel) (bool, error)
}

func (m SK8sOwnedResourceBaseManager) newOwnedModel(obj jsonutils.JSONObject) (IK8sOwnedResource, error) {
	model, err := db.NewModelObject(m.ownedManager)
	if err != nil {
		return nil, errors.Wrap(err, "db.NewModelObject")
	}
	if err := obj.Unmarshal(model); err != nil {
		return nil, errors.Wrap(err, "json marshal")
	}
	return model.(IK8sOwnedResource), nil
}

func (m SK8sOwnedResourceBaseManager) CustomizeFilterList(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*db.CustomizeListFilters, error) {
	input := new(api.ListInputOwner)
	if err := query.Unmarshal(input); err != nil {
		return nil, err
	}
	filters := db.NewCustomizeListFilters()
	if !input.ShouldDo() {
		return filters, nil
	}
	man := GetK8sResourceManagerByKind(input.OwnerKind)
	if man == nil {
		return filters, httperrors.NewNotFoundError("Not found owner_kind %s", input.OwnerKind)
	}
	ff := func(obj jsonutils.JSONObject) (bool, error) {
		model, err := m.newOwnedModel(obj)
		if err != nil {
			return false, errors.Wrap(err, "newOwnedModel")
		}
		clusterId, _ := obj.GetString("cluster_id")
		namespaceId, _ := obj.GetString("namespace_id")
		ownerModel, err := FetchClusterResourceByIdOrName(man.(IClusterModelManager), userCred, clusterId, namespaceId, input.OwnerName)
		if err != nil {
			return false, errors.Wrapf(err, "get %s/%s/%s/%s", clusterId, namespaceId, input.OwnerKind, input.OwnerName)
		}
		return model.IsOwnedBy(ownerModel)
	}
	filters.Append(ff)
	return filters, nil
}

type IPodOwnerModel interface {
	IClusterModel

	GetRawPods(cli *client.ClusterManager, obj runtime.Object) ([]*v1.Pod, error)
}

func IsPodOwner(owner IPodOwnerModel, pod *SPod) (bool, error) {
	ownerObj, err := GetK8sObject(owner)
	if err != nil {
		return false, errors.Wrap(err, "get owner k8s object")
	}
	cli, err := owner.GetClusterClient()
	if err != nil {
		return false, errors.Wrap(err, "get cluster client")
	}
	pods, err := owner.GetRawPods(cli, ownerObj)
	if err != nil {
		return false, errors.Wrap(err, "get owner raw pods")
	}
	p, err := GetK8sObject(pod)
	if err != nil {
		return false, errors.Wrap(err, "get k8s pod")
	}
	return IsObjectContains(p.(*v1.Pod), pods), nil
}

type IServiceOwnerModel interface {
	IClusterModel

	GetRawServices(cli *client.ClusterManager, obj runtime.Object) ([]*v1.Service, error)
}

func IsServiceOwner(owner IServiceOwnerModel, svc *SService) (bool, error) {
	ownerObj, err := GetK8sObject(owner)
	if err != nil {
		return false, errors.Wrap(err, "get owner k8s object")
	}
	cli, err := owner.GetClusterClient()
	if err != nil {
		return false, errors.Wrap(err, "get cluster client")
	}
	svcs, err := owner.GetRawServices(cli, ownerObj)
	if err != nil {
		return false, errors.Wrap(err, "get owner raw services")
	}
	obj, err := GetK8sObject(svc)
	if err != nil {
		return false, errors.Wrap(err, "get k8s service")
	}
	return IsObjectContains(obj.(*v1.Service), svcs), nil
}

func IsObjectContains(obj metav1.Object, objs interface{}) bool {
	objsV := reflect.ValueOf(objs)
	for i := 0; i < objsV.Len(); i++ {
		ov := objsV.Index(i).Interface().(metav1.Object)
		if obj.GetUID() == ov.GetUID() {
			return true
		}
	}
	return false
}

type UnstructuredResourceBase struct{}

func (_ UnstructuredResourceBase) GetUnstructuredObject(m model.IK8sModel) *unstructured.Unstructured {
	return m.GetK8sObject().(*unstructured.Unstructured)
}

func (res UnstructuredResourceBase) GetRawJSONObject(m model.IK8sModel) (jsonutils.JSONObject, error) {
	rawObj := res.GetUnstructuredObject(m)
	jsonBytes, err := rawObj.MarshalJSON()
	if err != nil {
		return nil, err
	}
	jsonObj, err := jsonutils.Parse(jsonBytes)
	if err != nil {
		return nil, errors.Wrapf(err, "parse json bytes %q", string(jsonBytes))
	}
	return jsonObj, nil
}

type IUnstructuredOutput interface {
	SetObjectMeta(meta api.ObjectMeta) *api.ObjectTypeMeta
	SetTypeMeta(meta api.TypeMeta) *api.ObjectTypeMeta
}

type IK8SUnstructuredModel interface {
	model.IK8sModel

	FillAPIObjectBySpec(rawObjSpec jsonutils.JSONObject, output IUnstructuredOutput) error
	FillAPIObjectByStatus(rawObjStatus jsonutils.JSONObject, output IUnstructuredOutput) error
}

func (res UnstructuredResourceBase) ConvertToAPIObject(m IK8SUnstructuredModel, output IUnstructuredOutput) error {
	objMeta, err := m.GetObjectMeta()
	if err != nil {
		return errors.Wrap(err, "GetObjectMeta")
	}
	output.SetObjectMeta(objMeta).SetTypeMeta(m.GetTypeMeta())
	jsonObj, err := res.GetRawJSONObject(m)
	if err != nil {
		return errors.Wrap(err, "get json object")
	}
	specObj, err := jsonObj.Get("spec")
	if err != nil {
		log.Errorf("Get spec object error: %v", err)
	} else {
		if err := m.FillAPIObjectBySpec(specObj, output); err != nil {
			log.Errorf("FillAPIObjectBySpec error: %v", err)
		}
	}
	statusObj, err := jsonObj.Get("status")
	if err != nil {
		log.Errorf("Get status object error: %v", err)
	} else {
		if err := m.FillAPIObjectByStatus(statusObj, output); err != nil {
			log.Errorf("FillAPIObjectByStatus error: %v", err)
		}
	}
	return nil
}
