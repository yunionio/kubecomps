package imported

import (
	"context"

	"k8s.io/client-go/rest"

	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
)

type IImportDriver interface {
	GetDistribution() string
	ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.ClusterCreateInput, config *rest.Config) error
	// GetClusterUsers query users resource from remote k8s cluster
	GetClusterUsers(cluster *models.SCluster, restCfg *rest.Config) ([]api.ClusterUser, error)
	// GetClusterUserGroups query groups resource from remote k8s cluster
	GetClusterUserGroups(cluster *models.SCluster, restCfg *rest.Config) ([]api.ClusterUserGroup, error)
}

type ICloudImportDriver interface {
	IImportDriver

	GetProvider() api.ProviderType
}
