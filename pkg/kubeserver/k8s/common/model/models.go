package model

import (
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/object"
)

type IModelManager interface {
	lockman.ILockedClass
	object.IObject

	GetK8sResourceInfo() K8sResourceInfo
	GetOwnerModel(userCred mcclient.TokenCredential, manager IModelManager, cluster ICluster, namespace string, nameOrId string) (IOwnerModel, error)
}

var (
	GetK8sModelManagerByKind func(kindName string) IModelManager
)
