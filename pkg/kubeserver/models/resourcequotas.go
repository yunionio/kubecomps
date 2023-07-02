package models

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
)

var (
	resourceQuotaManager *SResourceQuotaManager
	_                    IClusterModel = new(SResourceQuota)
)

func init() {
	GetResourceQuotaManager()
}

func GetResourceQuotaManager() *SResourceQuotaManager {
	if resourceQuotaManager == nil {
		resourceQuotaManager = NewK8sNamespaceModelManager(func() ISyncableManager {
			return &SResourceQuotaManager{
				SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
					SResourceQuota{},
					"resourcequotas_tbl",
					"resourcequota",
					"resourcequotas",
					api.ResourceNameResourceQuota,
					v1.GroupName,
					v1.SchemeGroupVersion.Version,
					api.KindNameResourceQuota,
					new(v1.LimitRange),
				),
			}
		}).(*SResourceQuotaManager)
	}
	return resourceQuotaManager
}

// +onecloud:swagger-gen-model-singular=resourcequota
// +onecloud:swagger-gen-model-plural=resourcequotas
type SResourceQuotaManager struct {
	SNamespaceResourceBaseManager
}

type SResourceQuota struct {
	SNamespaceResourceBase
}

func (m *SResourceQuotaManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewBadRequestError("Not support resourcequota create")
}

func (obj *SResourceQuota) GetDetails(ctx context.Context, cli *client.ClusterManager, base interface{}, k8sObj runtime.Object, isList bool) interface{} {
	rq := k8sObj.(*v1.ResourceQuota)
	detail := api.ResourceQuotaDetailV2{
		NamespaceResourceDetail: obj.SNamespaceResourceBase.GetDetails(ctx, cli, base, k8sObj, isList).(api.NamespaceResourceDetail),
		ResourceQuotaSpec:       rq.Spec,
	}
	if isList {
		return detail
	}
	statusList := make(map[v1.ResourceName]api.ResourceStatus)
	for key, value := range rq.Status.Hard {
		used := rq.Status.Used[key]
		statusList[key] = api.ResourceStatus{
			Used: used.String(),
			Hard: value.String(),
		}
	}
	detail.StatusList = statusList
	return detail
}

// TODO: support filter by namespace owner
