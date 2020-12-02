package models

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
)

var (
	pvManager *SPVManager
	_         IClusterModel = new(SPV)
)

func init() {
	GetPVManager()
}

func GetPVManager() *SPVManager {
	if pvManager == nil {
		pvManager = NewK8sModelManager(func() ISyncableManager {
			return &SPVManager{
				SClusterResourceBaseManager: NewClusterResourceBaseManager(
					SPV{},
					"persistentvolumes_tbl",
					"persistentvolume",
					"persistentvolumes",
					api.ResourceNamePersistentVolume,
					v1.GroupName,
					v1.SchemeGroupVersion.Version,
					api.KindNamePersistentVolume,
					new(v1.PersistentVolume),
				),
			}
		}).(*SPVManager)
	}
	return pvManager
}

// +onecloud:swagger-gen-model-singular=persistentvolume
// +onecloud:swagger-gen-model-plural=persistentvolumes
type SPVManager struct {
	SClusterResourceBaseManager
}

type SPV struct {
	SClusterResourceBase
}

func (obj *SPV) getPVCShortDesc(pv *v1.PersistentVolume) string {
	var claim string
	if pv.Spec.ClaimRef != nil {
		claim = pv.Spec.ClaimRef.Namespace + "/" + pv.Spec.ClaimRef.Namespace
	}
	return claim
}

func (obj *SPV) GetDetails(cli *client.ClusterManager, base interface{}, k8sObj runtime.Object, isList bool) interface{} {
	pv := k8sObj.(*v1.PersistentVolume)
	detail := api.PersistentVolumeDetailV2{
		ClusterResourceDetail: obj.SClusterResourceBase.GetDetails(cli, base, k8sObj, isList).(api.ClusterResourceDetail),
		Capacity:              pv.Spec.Capacity,
		AccessModes:           pv.Spec.AccessModes,
		ReclaimPolicy:         pv.Spec.PersistentVolumeReclaimPolicy,
		StorageClass:          pv.Spec.StorageClassName,
		Status:                pv.Status.Phase,
		Claim:                 obj.getPVCShortDesc(pv),
		Reason:                pv.Status.Reason,
		Message:               pv.Status.Message,
	}
	if isList {
		return detail
	}
	// todo: add pvc info
	return detail
}
