package models

import (
	"context"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	// "yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
	"yunion.io/x/kubecomps/pkg/kubeserver/models/logevent"
)

var (
	pvcManager *SPVCManager
	_          IClusterModel = new(SPVC)
)

func init() {
	GetPVCManager()
}

func GetPVCManager() *SPVCManager {
	if pvcManager == nil {
		pvcManager = NewK8sModelManager(func() ISyncableManager {
			return &SPVCManager{
				SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
					SPVC{},
					"persistentvolumeclaims_tbl",
					"persistentvolumeclaim",
					"persistentvolumeclaims",
					api.ResourceNamePersistentVolumeClaim,
					v1.GroupName,
					v1.SchemeGroupVersion.Version,
					api.KindNamePersistentVolumeClaim,
					new(v1.PersistentVolumeClaim),
				),
			}
		}).(*SPVCManager)
	}
	return pvcManager
}

// +onecloud:swagger-gen-model-singular=persistentvolumeclaim
// +onecloud:swagger-gen-model-plural=persistentvolumeclaims
type SPVCManager struct {
	SNamespaceResourceBaseManager
}

type SPVC struct {
	SNamespaceResourceBase
}

func (m *SPVCManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.PersistentVolumeClaimListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SNamespaceResourceBaseManager.ListItemFilter(ctx, q, userCred, &input.NamespaceResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SNamespaceResourceBaseManager.ListItemFilter")
	}

	// ceph pvc 可用时也为 Bound 状态，所以用状态过滤不准
	/*if input.Unused != nil {
		unused := *input.Unused
		if !unused {
			q = q.Equals("status", string(v1.ClaimBound))
		} else {
			q = q.NotEquals("status", string(v1.ClaimBound))
		}
	}*/
	return q, nil
}

/*func (m *SPVCManager) CustomizeFilterList(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*db.CustomizeListFilters, error) {
	input := new(api.PersistentVolumeClaimListInput)
	if err := query.Unmarshal(input); err != nil {
		return nil, err
	}
	filters := db.NewCustomizeListFilters()
	ff := func(obj jsonutils.JSONObject) (bool, error) {
		dbObj, err := db.NewModelObject(m)
		if err != nil {
			return false, errors.Wrap(err, "new pvc db model obect")
		}
		if err := obj.Unmarshal(dbObj); err != nil {
			return false, errors.Wrap(err, "unmarshal json object")
		}
		pvc := dbObj.(*SPVC)
		cli, err := pvc.GetClusterClient()
		if err != nil {
			return false, errors.Wrap(err, "get cluster client")
		}
		rawPvc, err := pvc.GetK8sObject()
		if err != nil {
			return false, errors.Wrap(err, "get raw k8s pvc")
		}
		if input.Unused != nil {
			mntPods, err := dbObj.(*SPVC).GetMountPodNames(cli, rawPvc.(*v1.PersistentVolumeClaim))
			if err != nil {
				return false, errors.Wrap(err, "get mount pods")
			}
			if *input.Unused {
				if len(mntPods) == 0 {
					return true, nil
				} else {
					return false, nil
				}
			} else {
				if len(mntPods) > 0 {
					return true, nil
				} else {
					return false, nil
				}
			}
		}
		return true, nil
	}
	filters.Append(ff)
	return filters, nil
}*/

func (m *SPVCManager) NewRemoteObjectForCreate(model IClusterModel, cli *client.ClusterManager, body jsonutils.JSONObject) (interface{}, error) {
	input := new(api.PersistentVolumeClaimCreateInput)
	body.Unmarshal(input)
	size := input.Size
	storageSize, err := resource.ParseQuantity(size)
	if err != nil {
		return nil, err
	}
	objMeta, err := input.ToObjectMeta(model.(api.INamespaceGetter))
	if err != nil {
		return nil, err
	}
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: objMeta,
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					"storage": storageSize,
				},
			},
			StorageClassName: &input.StorageClass,
		},
		Status: v1.PersistentVolumeClaimStatus{},
	}
	return pvc, nil
}

func (_ *SPVCManager) GetPVCVolumes(vols []v1.Volume) []v1.Volume {
	var pvcs []v1.Volume
	for _, vol := range vols {
		if vol.VolumeSource.PersistentVolumeClaim != nil {
			pvcs = append(pvcs, vol)
		}
	}
	return pvcs
}

func (obj *SPVC) getMountRawPods(cli *client.ClusterManager, pvc *v1.PersistentVolumeClaim) ([]*v1.Pod, error) {
	pods, err := GetPodManager().GetRawPodsByObjectNamespace(cli, pvc)
	if err != nil {
		return nil, err
	}
	mPods := make([]*v1.Pod, 0)
	for _, pod := range pods {
		pvcs := GetPVCManager().GetPVCVolumes(pod.Spec.Volumes)
		for _, pvc := range pvcs {
			if pvc.PersistentVolumeClaim.ClaimName == obj.GetName() {
				mPods = append(mPods, pod)
			}
		}
	}
	return mPods, nil
}

func (obj *SPVC) GetRawPods(cli *client.ClusterManager, rawObj runtime.Object) ([]*v1.Pod, error) {
	pvc := rawObj.(*v1.PersistentVolumeClaim)
	return obj.getMountRawPods(cli, pvc)
}

func (obj *SPVC) SetStatusByRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, extObj interface{}) error {
	curStatus := string(extObj.(*v1.PersistentVolumeClaim).Status.Phase)
	if obj.Status != curStatus {
		note := obj.ToEventNote(ctx, userCred, extObj)
		obj.Status = curStatus
		db.OpsLog.LogEvent(obj, curStatus, jsonutils.Marshal(note), userCred)
	}
	return nil
}

func (obj *SPVC) ToEventNote(ctx context.Context, userCred mcclient.TokenCredential, k8sObj interface{}) interface{} {
	return ToResourceEventNote(ctx, userCred, obj, k8sObj, func(domainId string, nsLabels map[string]string, detailObj interface{}) interface{} {
		detail := detailObj.(api.PersistentVolumeClaimDetail)
		return logevent.NewPVCNote(domainId, detail, nsLabels)
	})
}

func (obj *SPVC) GetMountPodNames(cli *client.ClusterManager, pvc *v1.PersistentVolumeClaim) ([]string, error) {
	pods, err := obj.getMountRawPods(cli, pvc)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0)
	for _, p := range pods {
		names = append(names, p.GetName())
	}
	return names, nil
}

func (obj *SPVC) GetDetails(cli *client.ClusterManager, base interface{}, k8sObj runtime.Object, isList bool) interface{} {
	pvc := k8sObj.(*v1.PersistentVolumeClaim)
	detail := api.PersistentVolumeClaimDetail{
		NamespaceResourceDetail: obj.SNamespaceResourceBase.GetDetails(cli, base, k8sObj, isList).(api.NamespaceResourceDetail),
		Status:                  string(pvc.Status.Phase),
		Volume:                  pvc.Spec.VolumeName,
		Capacity:                pvc.Status.Capacity,
		AccessModes:             pvc.Spec.AccessModes,
		StorageClass:            pvc.Spec.StorageClassName,
	}
	if podNames, err := obj.GetMountPodNames(cli, pvc); err != nil {
		log.Errorf("get mount pods error: %v", err)
	} else {
		detail.MountedBy = podNames
	}
	return detail
}
