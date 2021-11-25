package imported

import (
	"context"

	"k8s.io/client-go/rest"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	onecloudcli "yunion.io/x/kubecomps/pkg/utils/onecloud/client"
)

type baseDriver struct {
	distribution string
}

func newBaseDriver(distro string) *baseDriver {
	return &baseDriver{
		distribution: distro,
	}
}

func (d *baseDriver) GetDistribution() string {
	return d.distribution
}

func (d *baseDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.ClusterCreateInput, config *rest.Config) error {
	return nil
}

func (d *baseDriver) GetClusterUsers(cluster *models.SCluster, config *rest.Config) ([]api.ClusterUser, error) {
	return nil, nil
}

func (d *baseDriver) GetClusterUserGroups(cluster *models.SCluster, config *rest.Config) ([]api.ClusterUserGroup, error) {
	return nil, nil
}

type k8sBaseDriver struct {
	*baseDriver
}

func newK8sBaseDriver() *k8sBaseDriver {
	return &k8sBaseDriver{
		baseDriver: newBaseDriver(api.ImportClusterDistributionK8s),
	}
}

type cloudK8sBaseDriver struct {
	*k8sBaseDriver

	provider api.ProviderType
}

func newCloudK8sBaseDriver(provider api.ProviderType) *cloudK8sBaseDriver {
	return &cloudK8sBaseDriver{
		k8sBaseDriver: newK8sBaseDriver(),
		provider:      provider,
	}
}

func (d *cloudK8sBaseDriver) GetProvider() api.ProviderType {
	return d.provider
}

func (d *cloudK8sBaseDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.ClusterCreateInput, config *rest.Config) error {
	s, err := models.GetClusterManager().GetSession()
	if err != nil {
		return errors.Wrap(err, "get session")
	}
	// check cloudregion
	if input.CloudregionId == "" {
		return httperrors.NewNotEmptyError("cloudregion_id is empty")
	}
	cli := onecloudcli.NewClientSets(s)
	obj, err := cli.Cloudregions().GetDetails(input.CloudregionId)
	if err != nil {
		return errors.Wrapf(err, "get cloudregion_id %s", input.CloudregionId)
	}
	input.CloudregionId = obj.Id
	return nil
}
