package model

import (
	"k8s.io/apimachinery/pkg/version"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/object"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type IModelManager interface {
	lockman.ILockedClass
	object.IObject

	GetK8sResourceInfo(version *version.Info) K8sResourceInfo
	GetOwnerModel(userCred mcclient.TokenCredential, manager IModelManager, cluster ICluster, namespace string, nameOrId string) (IOwnerModel, error)
}

var (
	GetK8sModelManagerByKind func(kindName string) IModelManager
)
