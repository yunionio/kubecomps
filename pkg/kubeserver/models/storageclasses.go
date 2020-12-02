package models

import (
	"context"
	// "strings"

	// corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers"
)

const (
	IsDefaultStorageClassAnnotation     = "storageclass.kubernetes.io/is-default-class"
	betaIsDefaultStorageClassAnnotation = "storageclass.beta.kubernetes.io/is-default-class"
)

var (
	storageClassManager *SStorageClassManager
	_                   IClusterModel = new(SStorageClass)
)

func init() {
	GetStorageClassManager()
}

func GetStorageClassManager() *SStorageClassManager {
	if storageClassManager == nil {
		storageClassManager = NewK8sModelManager(func() ISyncableManager {
			return &SStorageClassManager{
				SClusterResourceBaseManager: NewClusterResourceBaseManager(
					SStorageClass{},
					"storageclasses_tbl",
					"storageclass",
					"storageclasses",
					api.ResourceNameStorageClass,
					v1.GroupName,
					v1.SchemeGroupVersion.Version,
					api.KindNameStorageClass,
					new(v1.StorageClass),
				),
				driverManager: drivers.NewDriverManager(""),
			}
		}).(*SStorageClassManager)
	}
	return storageClassManager
}

// +onecloud:swagger-gen-model-singular=storageclass
// +onecloud:swagger-gen-model-plural=storageclasses
type SStorageClassManager struct {
	SClusterResourceBaseManager
	driverManager *drivers.DriverManager
}

type SStorageClass struct {
	SClusterResourceBase
}

type IStorageClassDriver interface {
	ConnectionTest(userCred mcclient.TokenCredential, cli *client.ClusterManager, input *api.StorageClassCreateInput) (*api.StorageClassTestResult, error)
	ValidateCreateData(userCred mcclient.TokenCredential, cli *client.ClusterManager, input *api.StorageClassCreateInput) (*api.StorageClassCreateInput, error)
	ToStorageClassParams(input *api.StorageClassCreateInput) (map[string]string, error)
}

func (m *SStorageClassManager) RegisterDriver(provisioner string, driver IStorageClassDriver) {
	if err := m.driverManager.Register(driver, provisioner); err != nil {
		panic(errors.Wrapf(err, "storageclass register driver %s", provisioner))
	}
}

func (m *SStorageClassManager) GetDriver(provisioner string) (IStorageClassDriver, error) {
	drv, err := m.driverManager.Get(provisioner)
	if err != nil {
		if errors.Cause(err) == drivers.ErrDriverNotFound {
			return nil, httperrors.NewNotFoundError("storageclass get %s driver", provisioner)
		}
		return nil, err
	}
	return drv.(IStorageClassDriver), nil
}

func (m *SStorageClassManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.StorageClassCreateInput) (*api.StorageClassCreateInput, error) {
	cinput, err := m.SClusterResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, &input.ClusterResourceCreateInput)
	if err != nil {
		return input, err
	}
	input.ClusterResourceCreateInput = *cinput
	if input.Provisioner == "" {
		return nil, httperrors.NewNotEmptyError("provisioner is empty")
	}
	drv, err := m.GetDriver(input.Provisioner)
	if err != nil {
		return nil, err
	}
	cli, err := GetClusterClient(input.ClusterId)
	if err != nil {
		return nil, errors.Wrapf(err, "get cluster client %s", input.ClusterId)
	}
	return drv.ValidateCreateData(userCred, cli, input)
}

func (m *SStorageClassManager) NewRemoteObjectForCreate(model IClusterModel, cli *client.ClusterManager, body jsonutils.JSONObject) (interface{}, error) {
	input := new(api.StorageClassCreateInput)
	body.Unmarshal(input)
	drv, err := m.GetDriver(input.Provisioner)
	if err != nil {
		return nil, err
	}
	params, err := drv.ToStorageClassParams(input)
	if err != nil {
		return nil, err
	}
	objMeta := input.ToObjectMeta()
	return &v1.StorageClass{
		ObjectMeta:        objMeta,
		Provisioner:       input.Provisioner,
		Parameters:        params,
		ReclaimPolicy:     input.ReclaimPolicy,
		MountOptions:      input.MountOptions,
		VolumeBindingMode: input.VolumeBindingMode,
		AllowedTopologies: input.AllowedTopologies,
	}, nil
}

func (m *SStorageClass) GetDetails(cli *client.ClusterManager, base interface{}, k8sObj runtime.Object, isList bool) interface{} {
	sc := k8sObj.(*v1.StorageClass)
	isDefault := false
	if _, ok := sc.Annotations[IsDefaultStorageClassAnnotation]; ok {
		isDefault = true
	}
	if _, ok := sc.Annotations[betaIsDefaultStorageClassAnnotation]; ok {
		isDefault = true
	}
	detail := api.StorageClassDetailV2{
		ClusterResourceDetail: m.SClusterResourceBase.GetDetails(cli, base, k8sObj, isList).(api.ClusterResourceDetail),
		Provisioner:           sc.Provisioner,
		Parameters:            sc.Parameters,
		IsDefault:             isDefault,
	}
	if isList {
		return detail
	}
	return detail
}

func (m *SStorageClassManager) PerformConnectionTest(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.StorageClassCreateInput) (*api.StorageClassTestResult, error) {
	drv, err := m.GetDriver(input.Provisioner)
	if err != nil {
		return nil, err
	}
	cli, err := GetClusterClient(input.ClusterId)
	if err != nil {
		return nil, errors.Wrapf(err, "get cluster client %s", input.ClusterId)
	}
	return drv.ConnectionTest(userCred, cli, input)
}

func (m *SStorageClassManager) GetRawStorageClasses(cli *client.ClusterManager) ([]*v1.StorageClass, error) {
	return cli.GetHandler().GetIndexer().StorageClassLister().List(labels.Everything())
}

func (obj *SStorageClass) AllowPerformSetDefault(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, obj, "set-default")
}

func (obj *SStorageClass) PerformSetDefault(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	cli, err := obj.GetClusterClient()
	if err != nil {
		return nil, err
	}
	scList, err := storageClassManager.GetRawStorageClasses(cli)
	if err != nil {
		return nil, err
	}
	k8sCli := cli.GetClientset()
	for _, sc := range scList {
		_, hasDefault := sc.Annotations[IsDefaultStorageClassAnnotation]
		_, hasBeta := sc.Annotations[betaIsDefaultStorageClassAnnotation]
		if sc.Annotations == nil {
			sc.Annotations = make(map[string]string)
		}
		if sc.Name == obj.GetName() || hasDefault || hasBeta {
			delete(sc.Annotations, IsDefaultStorageClassAnnotation)
			delete(sc.Annotations, betaIsDefaultStorageClassAnnotation)
			if sc.Name == obj.GetName() {
				sc.Annotations[IsDefaultStorageClassAnnotation] = "true"
			}
			_, err := k8sCli.StorageV1().StorageClasses().Update(ctx, sc, metav1.UpdateOptions{})
			if err != nil {
				return nil, err
			}
		}
	}
	return nil, nil
}
