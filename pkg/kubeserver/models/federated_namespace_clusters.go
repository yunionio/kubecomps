package models

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
)

var (
	FedNamespaceClusterManager *SFedNamespaceClusterManager
	_                          IFedJointClusterModel = new(SFedNamespaceCluster)
)

func init() {
	db.InitManager(func() {
		FedNamespaceClusterManager = NewFedJointManager(func() db.IJointModelManager {
			return &SFedNamespaceClusterManager{
				SFedJointClusterManager: NewFedJointClusterManager(
					SFedNamespaceCluster{},
					"federatednamespaceclusters_tbl",
					"federatednamespacecluster",
					"federatednamespaceclusters",
					GetFedNamespaceManager(),
					GetNamespaceManager(),
				),
			}
		}).(*SFedNamespaceClusterManager)
		GetFedNamespaceManager().SetJointModelManager(FedNamespaceClusterManager)
		RegisterFedJointClusterManager(GetFedNamespaceManager(), FedNamespaceClusterManager)
	})
}

// +onecloud:swagger-gen-model-singular=federatednamespacecluster
// +onecloud:swagger-gen-model-plural=federatednamespaceclusters
type SFedNamespaceClusterManager struct {
	SFedJointClusterManager
}

type SFedNamespaceCluster struct {
	SFedJointCluster
}

func (m *SFedNamespaceClusterManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.FederatedNamespaceClusterListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SFedJointClusterManager.ListItemFilter(ctx, q, userCred, &input.FedJointClusterListInput)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (obj *SFedNamespaceCluster) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, obj)
}

func (obj *SFedNamespaceCluster) GetFedNamespace() (*SFedNamespace, error) {
	fedObj, err := GetFedResAPI().JointResAPI().FetchFedResourceModel(obj)
	if err != nil {
		return nil, errors.Wrap(err, "get federated namespace")
	}
	return fedObj.(*SFedNamespace), nil
}

func (obj *SFedNamespaceCluster) GetDetails(base api.FedJointClusterResourceDetails, isList bool) interface{} {
	out := api.FederatedNamespaceClusterDetails{
		FedJointClusterResourceDetails: obj.SFedJointCluster.GetDetails(base, isList).(api.FedJointClusterResourceDetails),
	}
	return out
}

func (obj *SFedNamespaceCluster) GetK8sResource() (runtime.Object, error) {
	fedNs, err := obj.GetFedNamespace()
	if err != nil {
		return nil, errors.Wrap(err, "get federated namespace")
	}
	ns := &corev1.Namespace{
		ObjectMeta: fedNs.GetK8sObjectMeta(),
		Spec:       fedNs.Spec.Template.Spec,
	}
	return ns, nil
}

func (obj *SFedNamespaceCluster) GetResourceCreateData(ctx context.Context, userCred mcclient.TokenCredential, fObj IFedModel, base api.NamespaceResourceCreateInput) (jsonutils.JSONObject, error) {
	input := api.NamespaceCreateInputV2{
		ClusterResourceCreateInput: base.ClusterResourceCreateInput,
	}
	return input.JSON(input), nil
}

func (obj *SFedNamespaceCluster) GetResourceUpdateData(ctx context.Context, userCred mcclient.TokenCredential, fObj IFedModel, resObj IClusterModel, base api.NamespaceResourceUpdateInput) (jsonutils.JSONObject, error) {
	// TODO: namespace should update spec
	return base.JSON(base), nil
}
