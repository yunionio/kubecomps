package clusters

import (
	"context"

	"k8s.io/client-go/rest"

	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
)

type importK8sDriver struct{}

func newImportK8sDriver() iImportDriver {
	return new(importK8sDriver)
}

func (d *importK8sDriver) GetDistribution() string {
	return api.ImportClusterDistributionK8s
}

func (d *importK8sDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.ClusterCreateInput, config *rest.Config) error {
	return nil
}

func (d *importK8sDriver) GetClusterUsers(cluster *models.SCluster, config *rest.Config) ([]api.ClusterUser, error) {
	return nil, nil
}

func (d *importK8sDriver) GetClusterUserGroups(cluster *models.SCluster, config *rest.Config) ([]api.ClusterUserGroup, error) {
	return nil, nil
}
