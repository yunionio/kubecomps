package models

import (
	"context"
	"database/sql"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
)

var (
	namespaceManager *SNamespaceManager
	_                IClusterModelManager = new(SNamespaceManager)
	_                IClusterModel        = new(SNamespace)
)

func init() {
	GetNamespaceManager()
}

func GetNamespaceManager() *SNamespaceManager {
	if namespaceManager == nil {
		namespaceManager = newK8sModelManager(func() ISyncableManager {
			return &SNamespaceManager{
				SClusterResourceBaseManager: NewClusterResourceBaseManager(
					SNamespace{},
					"namespaces_tbl",
					"namespace",
					"namespaces",
					api.ResourceNameNamespace,
					v1.GroupName,
					v1.SchemeGroupVersion.Version,
					api.KindNameNamespace,
					&v1.Namespace{}),
			}
		}).(*SNamespaceManager)
		namespaceManager.RegisterFederatedManager(GetFedNamespaceManager())
		// namespaces should sync after nodes synced
		GetNodeManager().AddSubManager(namespaceManager)
	}
	return namespaceManager
}

type SNamespaceManager struct {
	SClusterResourceBaseManager
	SFederatedManagedResourceBaseManager
}

type SNamespace struct {
	SClusterResourceBase
	SFederatedManagedResourceBase
}

func (m *SNamespaceManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.NamespaceCreateInputV2) (*api.NamespaceCreateInputV2, error) {
	cData, err := m.SClusterResourceBaseManager.ValidateCreateData(ctx, userCred, ownerCred, query, &data.ClusterResourceCreateInput)
	if err != nil {
		return nil, err
	}
	data.ClusterResourceCreateInput = *cData
	return data, nil
}

func (obj *SNamespace) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if err := obj.SClusterResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data); err != nil {
		return err
	}
	return nil
}

func (res *SNamespace) SetCluster(userCred mcclient.TokenCredential, cls *SCluster) {
	res.SClusterResourceBase.SetCluster(userCred, cls)
}

func (m *SNamespaceManager) NewRemoteObjectForCreate(obj IClusterModel, _ *client.ClusterManager, data jsonutils.JSONObject) (interface{}, error) {
	input := new(api.NamespaceCreateInputV2)
	if err := data.Unmarshal(input); err != nil {
		return nil, err
	}
	objMeta := input.ToObjectMeta()
	return &v1.Namespace{
		ObjectMeta: objMeta,
	}, nil
}

func (m *SNamespaceManager) EnsureNamespace(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, cluster *SCluster, data *api.NamespaceCreateInputV2) (*SNamespace, error) {
	data.ClusterId = cluster.GetId()
	nsObj, err := m.GetByIdOrName(userCred, cluster.GetId(), data.Name)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			nsModelObj, err := db.DoCreate(m, ctx, userCred, nil, data.JSON(data), ownerCred)
			if err != nil {
				return nil, errors.Wrap(err, "create namespace")
			}
			nsObj = nsModelObj.(IClusterModel)
		} else {
			return nil, errors.Wrapf(err, "get namespace %s", data.Name)
		}
	}
	ns := nsObj.(*SNamespace)
	if err := ns.DoSync(ctx, userCred); err != nil {
		return nil, errors.Wrap(err, "sync to kubernetes cluster")
	}
	return ns, nil
}

func (ns *SNamespace) DoSync(ctx context.Context, userCred mcclient.TokenCredential) error {
	cluster, err := ns.GetCluster()
	if err != nil {
		return errors.Wrapf(err, "get namespace %s cluster", ns.GetName())
	}
	if err := EnsureNamespace(cluster, ns.GetName()); err != nil {
		return errors.Wrapf(err, "ensure namespace %s", ns.GetName())
	}
	rNs, err := GetK8sObject(ns)
	if err != nil {
		return errors.Wrap(err, "get namespace k8s object")
	}
	return SyncUpdatedClusterResource(ctx, userCred, namespaceManager, ns, rNs)
}

func (m *SNamespaceManager) NewFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, obj interface{}) (IClusterModel, error) {
	nsObj, err := m.SClusterResourceBaseManager.NewFromRemoteObject(ctx, userCred, cluster, obj)
	if err != nil {
		return nil, err
	}
	return nsObj, nil
}

func (ns *SNamespace) UpdateFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, extObj interface{}) error {
	if err := ns.SClusterResourceBase.UpdateFromRemoteObject(ctx, userCred, extObj); err != nil {
		return err
	}
	return nil
}

func (ns *SNamespace) SetStatusByRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, extObj interface{}) error {
	k8sNsStatus := string(extObj.(*v1.Namespace).Status.Phase)
	ns.Status = k8sNsStatus
	return nil
}

func (m *SNamespaceManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.NamespaceListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SClusterResourceBaseManager.ListItemFilter(ctx, q, userCred, &input.ClusterResourceListInput)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (m *SNamespace) GetEvents() ([]*api.Event, error) {
	cli, err := m.GetClusterClient()
	if err != nil {
		return nil, err
	}
	return GetEventManager().GetNamespaceEvents(cli, m.GetName())
}

func (m *SNamespace) GetDetails(cli *client.ClusterManager, base interface{}, k8sObj runtime.Object, isList bool) interface{} {
	detail := m.SClusterResourceBase.GetDetails(cli, base, k8sObj, isList).(api.ClusterResourceDetail)
	out := api.NamespaceDetailV2{
		ClusterResourceDetail: detail,
	}
	return out
}

func (ns *SNamespace) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	if err := ns.SClusterResourceBase.Delete(ctx, userCred); err != nil {
		cluster, err := ns.GetCluster()
		if err != nil {
			return errors.Wrap(err, "get cluster")
		}
		if cluster.IsSystem {
			if !userCred.HasSystemAdminPrivilege() {
				return httperrors.NewForbiddenError("Not system admin")
			}
		}
		return err
	}
	for _, man := range []IClusterModelManager{
		GetReleaseManager(),
		GetPodManager(),
	} {
		q := man.Query()
		q.Equals("namespace_id", ns.GetId())
		cnt, err := q.CountWithError()
		if err != nil {
			return errors.Wrapf(err, "check %s namespace resource count", man.KeywordPlural())
		}
		if cnt != 0 {
			return httperrors.NewNotAcceptableError("%s has %d resource in namespace %s", man.KeywordPlural(), cnt, ns.GetName())
		}
	}
	return nil
}
