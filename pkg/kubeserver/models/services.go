package models

import (
	"context"
	"reflect"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/apis/core/validation"
	"yunion.io/x/log"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
)

var (
	serviceManager *SServiceManager
	_              IPodOwnerModel = new(SService)
)

func init() {
	GetServiceManager()
}

func GetServiceManager() *SServiceManager {
	if serviceManager == nil {
		serviceManager = NewK8sNamespaceModelManager(func() ISyncableManager {
			return &SServiceManager{
				SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
					SService{},
					"services_tbl",
					"k8s_service",
					"k8s_services",
					api.ResourceNameService,
					v1.GroupName,
					v1.SchemeGroupVersion.Version,
					api.KindNameService,
					new(v1.Service),
				),
			}
		}).(*SServiceManager)
	}
	return serviceManager
}

type SServiceManager struct {
	SNamespaceResourceBaseManager
}

type SService struct {
	SNamespaceResourceBase
}

func (m *SServiceManager) ValidateService(svc *v1.Service) error {
	return ValidateCreateK8sObject(svc, new(core.Service), func(out interface{}) field.ErrorList {
		return validation.ValidateService(out.(*core.Service), true)
		// return validation.ValidateObjectMeta(&svc.ObjectMeta, true, validation.ValidateServiceName, field.NewPath("metadata"))
	})
}

func (m *SServiceManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.ServiceCreateInput) (*api.ServiceCreateInput, error) {
	nInput, err := m.SNamespaceResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, &data.NamespaceResourceCreateInput)
	if err != nil {
		return nil, err
	}
	data.NamespaceResourceCreateInput = *nInput
	return data, nil
}

func (m *SServiceManager) NewRemoteObjectForCreate(model IClusterModel, cli *client.ClusterManager, data jsonutils.JSONObject) (interface{}, error) {
	input := new(api.ServiceCreateInput)
	data.Unmarshal(input)
	objMeta, err := input.ToObjectMeta(model.(api.INamespaceGetter))
	if err != nil {
		return nil, err
	}
	return GetServiceFromOption(&objMeta, &input.ServiceCreateOption), nil
}

func (m *SServiceManager) GetRawServicesByMatchLabels(cli *client.ClusterManager, namespace string, matchLabels map[string]string) ([]*v1.Service, error) {
	indexer := cli.GetHandler().GetIndexer()
	svcs, err := indexer.ServiceLister().Services(namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	ret := make([]*v1.Service, 0)
	for _, svc := range svcs {
		if reflect.DeepEqual(svc.Spec.Selector, matchLabels) {
			ret = append(ret, svc)
		}
	}
	return ret, nil
}

func (obj *SService) IsOwnedBy(ownerModel IClusterModel) (bool, error) {
	return IsServiceOwner(ownerModel.(IServiceOwnerModel), obj)
}

func (obj *SService) GetDetails(
	cli *client.ClusterManager,
	base interface{},
	k8sObj runtime.Object,
	isList bool,
) interface{} {
	svc := k8sObj.(*v1.Service)
	nodes, err := cli.GetClientset().CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Warningf("list nodes for service: %v", err)
	}
	detail := api.ServiceDetailV2{
		NamespaceResourceDetail: obj.SNamespaceResourceBase.GetDetails(cli, base, k8sObj, isList).(api.NamespaceResourceDetail),
		InternalEndpoint:        GetInternalEndpoint(nodes.Items, svc.Name, svc.Namespace, svc.Spec.Ports),
		ExternalEndpoints:       GetExternalEndpoints(svc),
		Selector:                svc.Spec.Selector,
		Type:                    svc.Spec.Type,
		ClusterIP:               svc.Spec.ClusterIP,
	}
	if isList {
		return detail
	}
	detail.SessionAffinity = svc.Spec.SessionAffinity
	return detail
}

func (obj *SService) GetRawPods(cli *client.ClusterManager, rawObj runtime.Object) ([]*v1.Pod, error) {
	svc := rawObj.(*v1.Service)
	selector := labels.SelectorFromSet(svc.Spec.Selector)
	return GetPodManager().GetRawPodsBySelector(cli, svc.GetNamespace(), selector)
}
