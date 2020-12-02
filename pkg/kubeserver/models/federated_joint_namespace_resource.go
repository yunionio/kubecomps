package models

import (
	"context"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
)

type IFedNamespaceJointClusterManager interface {
	IFedJointClusterManager
}

type IFedNamespaceJointClusterModel interface {
	IFedJointClusterModel
}

type SFedNamespaceJointClusterManager struct {
	SFedJointClusterManager
}

type SFederatedNamespaceJointCluster struct {
	SFedJointCluster
}

func NewFedNamespaceJointClusterManager(
	dt interface{}, tableName string,
	keyword string, keywordPlural string,
	master IFedModelManager,
	resourceMan IClusterModelManager,
) SFedNamespaceJointClusterManager {
	return SFedNamespaceJointClusterManager{
		SFedJointClusterManager: NewFedJointClusterManager(dt, tableName, keyword, keywordPlural, master, resourceMan),
	}
}

func (m *SFedNamespaceJointClusterManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.FedNamespaceJointClusterListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SFedJointClusterManager.ListItemFilter(ctx, q, userCred, &input.FedJointClusterListInput)
	if err != nil {
		return nil, err
	}
	if input.FederatedNamespaceId != "" {
		fedNsObj, err := GetFedNamespaceManager().FetchByIdOrName(userCred, input.FederatedNamespaceId)
		if err != nil {
			return nil, errors.Wrap(err, "Get federatednamespace")
		}
		masterSq := m.GetFedManager().Query("id").Equals("federatednamespace_id", fedNsObj.GetId()).SubQuery()
		q = q.In("federatedresource_id", masterSq)
	}
	return q, nil
}

func (obj *SFederatedNamespaceJointCluster) GetClusterNamespace(userCred mcclient.TokenCredential, clusterId string, namespace string) (*SNamespace, error) {
	nsObj, err := GetNamespaceManager().GetByName(userCred, clusterId, namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "get cluster %s namespace %s", clusterId, namespace)
	}
	return nsObj.(*SNamespace), nil
}

/*
 * func (obj *SFederatedNamespaceJointCluster) GetDetails(base interface{}, isList bool) interface{} {
 *     out := api.FedNamespaceJointClusterResourceDetails{
 *         FedJointClusterResourceDetails: base.(api.FedJointClusterResourceDetails),
 *     }
 * }
 */
