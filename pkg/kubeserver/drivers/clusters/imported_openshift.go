package clusters

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"
	configv1client "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	userv1client "github.com/openshift/client-go/user/clientset/versioned/typed/user/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
)

type importOpenshiftDriver struct{}

func newImportOpenshiftDriver() iImportDriver {
	return new(importOpenshiftDriver)
}

func (d *importOpenshiftDriver) GetDistribution() string {
	return api.ImportClusterDistributionOpenshift
}

func (d *importOpenshiftDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.ClusterCreateInput, config *rest.Config) error {
	ocCli, err := configv1client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "new openshift client")
	}
	var clusterOperator *configv1.ClusterOperator
	clusterOperator, serverErr := ocCli.ClusterOperators().Get(context.Background(), "openshift-apiserver", metav1.GetOptions{})
	if serverErr != nil {
		switch {
		case kerrors.IsForbidden(serverErr), kerrors.IsNotFound(serverErr):
			return errors.Wrapf(err, "OpenShift Version not found (must be logged in to cluster as admin): %v")
		}
		return errors.Wrap(err, "get openshift apiserver")
	}
	info := &input.ImportData.DistributionInfo
	for _, ver := range clusterOperator.Status.Versions {
		if ver.Name == "operator" {
			// openshift-apiserver does not report version,
			// clusteroperator/openshift-apiserver does, and only version number
			info.Version = ver.Version
		}
	}
	return nil
}

func (d *importOpenshiftDriver) GetClusterUsers(cluster *models.SCluster, config *rest.Config) ([]api.ClusterUser, error) {
	userCli, err := userv1client.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "new user client")
	}
	userList, err := userCli.Users().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	ret := make([]api.ClusterUser, 0)
	for _, user := range userList.Items {
		ret = append(ret, api.ClusterUser{
			Name:       user.Name,
			FullName:   user.FullName,
			Identities: user.Identities,
			Groups:     user.Groups,
		})
	}
	return ret, nil
}

func (d *importOpenshiftDriver) GetClusterUserGroups(cluster *models.SCluster, config *rest.Config) ([]api.ClusterUserGroup, error) {
	userCli, err := userv1client.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "new user client")
	}
	groupList, err := userCli.Groups().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	ret := make([]api.ClusterUserGroup, 0)
	for _, grp := range groupList.Items {
		ret = append(ret, api.ClusterUserGroup{
			Name:  grp.Name,
			Users: api.OptionalNames(grp.Users),
		})
	}
	return ret, nil
}
