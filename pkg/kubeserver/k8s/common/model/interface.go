package model

import (
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/object"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
)

type K8sResourceInfo struct {
	ResourceName string
	Group        string
	Version      string
	KindName     string
	Object       runtime.Object
}

type IK8sModelManager interface {
	IModelManager

	Factory() *SK8sObjectFactory
	KeywordPlural() string
	GetQuery(cluster ICluster) IQuery
	GetOrderFields() OrderFields
	RegisterOrderFields(fields ...IOrderField)
}

type IK8sModel interface {
	lockman.ILockedObject
	object.IObject

	GetName() string
	GetNamespace() string
	KeywordPlural() string

	GetModelManager() IK8sModelManager
	SetModelManager(manager IK8sModelManager, model IK8sModel) IK8sModel

	GetCluster() ICluster
	SetCluster(cluster ICluster) IK8sModel

	SetK8sObject(runtime.Object) IK8sModel
	GetK8sObject() runtime.Object

	GetTypeMeta() api.TypeMeta

	IOwnerModel
}

type IOwnerModel interface {
	GetObjectMeta() (api.ObjectMeta, error)
}
