package models

import (
	"context"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
)

var (
	ServiceAccountManager *SServiceAccountManager
	_                     IClusterModel = new(SServiceAccount)
)

func init() {
	ServiceAccountManager = NewK8sNamespaceModelManager(func() ISyncableManager {
		return &SServiceAccountManager{
			SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
				SServiceAccount{},
				"serviceaccounts_tbl",
				"serviceaccount",
				"serviceaccounts",
				api.ResourceNameServiceAccount,
				v1.GroupName,
				v1.SchemeGroupVersion.Version,
				api.KindNameServiceAccount,
				new(v1.ServiceAccount),
			),
		}
	}).(*SServiceAccountManager)
}

// +onecloud:swagger-gen-model-singular=serviceaccount
// +onecloud:swagger-gen-model-plural=serviceaccounts
type SServiceAccountManager struct {
	SNamespaceResourceBaseManager
}

type SServiceAccount struct {
	SNamespaceResourceBase
}

func (m *SServiceAccountManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewBadRequestError("Not support serviceaccount create")
}

func (obj *SServiceAccount) GetDetails(cli *client.ClusterManager, base interface{}, k8sObj runtime.Object, isList bool) interface{} {
	sa := k8sObj.(*v1.ServiceAccount)
	return api.ServiceAccountDetail{
		NamespaceResourceDetail:      obj.SNamespaceResourceBase.GetDetails(cli, base, k8sObj, isList).(api.NamespaceResourceDetail),
		Secrets:                      sa.Secrets,
		ImagePullSecrets:             sa.ImagePullSecrets,
		AutomountServiceAccountToken: sa.AutomountServiceAccountToken,
	}
}
