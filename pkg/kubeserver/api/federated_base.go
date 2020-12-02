package api

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	FederatedResourceStatusActive     = "active"
	FederatedResourceStatusUpdateFail = "update_fail"
	FederatedResourceStatusSyncing    = "syncing"
	FedreatedResourceStatusSyncFail   = "sync_fail"
)

type FederatedResourceCreateInput struct {
	K8sResourceCreateInput
}

type FederatedResourceJointClusterInput struct {
	apis.Meta
	ClusterId string `json:"cluster_id"`
}

type FederatedResourceListInput struct {
	apis.StatusDomainLevelResourceListInput
}

type FederatedNamespaceResourceListInput struct {
	FederatedResourceListInput
	FederatednamespaceId string `json:"federatednamespace_id"`
	// swagger:ignore
	// Deprecated
	Federatednamespace string `json:"federatednamespace" yunion-deprecated-by:"federatednamespace_id"`
}

type FederatedNamespaceResourceCreateInput struct {
	FederatedResourceCreateInput
	FederatednamespaceId string `json:"federatednamespace_id"`
	Federatednamespace   string `json:"-"`
}

func (input FederatedNamespaceResourceCreateInput) ToObjectMeta(namespace string) metav1.ObjectMeta {
	objMeta := input.FederatedResourceCreateInput.ToObjectMeta()
	objMeta.Namespace = namespace
	return objMeta
}

type FederatedResourceDetails struct {
	apis.StatusDomainLevelResourceDetails
	Placement    FederatedPlacement `json:"placement"`
	ClusterCount *int               `json:"cluster_count"`
}

type FederatedNamespaceResourceDetails struct {
	FederatedResourceDetails

	Federatednamespace string `json:"federatednamespace"`
}

type FederatedPlacement struct {
	Clusters []FederatedJointCluster `json:"clusters"`
}

type FederatedJointCluster struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type FedJointClusterResourceDetails struct {
	apis.JointResourceBaseDetails
	Cluster                  string `json:"cluster"`
	FederatedResource        string `json:"federatedresource"`
	FederatedResourceKeyword string `json:"federatedresource_keyword"`
	NamespaceId              string `json:"namespace_id"`
	Namespace                string `json:"namespace"`
	Resource                 string `json:"resource"`
	ResourceKeyword          string `json:"resource_keyword"`
	ResourceStatus           string `json:"resource_status"`
}

type FedNamespaceJointClusterResourceDetails struct {
	FedJointClusterResourceDetails
	FederatedNamespace string `json:"federatednamespace"`
}

type FedResourceUpdateInput struct {
	apis.StatusDomainLevelResourceBaseUpdateInput
}

type FedNamespaceResourceUpdateInput struct {
	FedResourceUpdateInput
}
