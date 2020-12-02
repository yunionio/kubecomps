package machines

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/machines/userdata"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	"yunion.io/x/kubecomps/pkg/kubeserver/models/manager"
	"yunion.io/x/kubecomps/pkg/kubeserver/options"
	"yunion.io/x/kubecomps/pkg/utils/certificates"
)

const (
	DEFAULT_DOCKER_GRAPH_DIR = "/opt/docker"
)

type sClusterAPIBaseDriver struct {
	*sBaseDriver
}

func newClusterAPIBaseDriver() *sClusterAPIBaseDriver {
	return &sClusterAPIBaseDriver{
		sBaseDriver: newBaseDriver(),
	}
}

func (d *sClusterAPIBaseDriver) UseClusterAPI() bool {
	return true
}

func (d *sClusterAPIBaseDriver) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, cluster *models.SCluster, machine *models.SMachine, data *jsonutils.JSONDict) error {
	return nil
}

func getUserDataBaseConfigure(session *mcclient.ClientSession, cluster *models.SCluster, machine *models.SMachine) userdata.BaseConfigure {
	o := options.Options
	schedulerUrl, err := session.GetServiceURL("scheduler", "internalURL")
	if err != nil {
		log.Errorf("Get internal scheduler endpoint error: %v", err)
	}
	return userdata.BaseConfigure{
		DockerConfigure: userdata.DockerConfigure{
			DockerGraphDir: DEFAULT_DOCKER_GRAPH_DIR,
		},
		OnecloudConfigure: userdata.OnecloudConfigure{
			AuthURL:           o.AuthURL,
			AdminUser:         o.AdminUser,
			AdminPassword:     o.AdminPassword,
			AdminProject:      o.AdminProject,
			Region:            o.Region,
			Cluster:           cluster.Name,
			SchedulerEndpoint: schedulerUrl,
		},
	}
}

func (d *sClusterAPIBaseDriver) getUserData(session *mcclient.ClientSession, machine *models.SMachine, data *api.MachinePrepareInput) (string, error) {
	var userData string
	var err error

	caCertHash, err := certificates.GenerateCertificateHash(data.CAKeyPair.Cert)
	if err != nil {
		return "", err
	}

	cluster, err := machine.GetCluster()
	if err != nil {
		return "", err
	}

	baseConfigure := getUserDataBaseConfigure(session, cluster, machine)

	// apply userdata values based on the role of the machine
	switch data.Role {
	case api.RoleTypeControlplane:
		if data.BootstrapToken != "" {
			log.Infof("Allow machine %q to join control plane for cluster %q", machine.Name, cluster.Name)
			userData, err = userdata.JoinControlPlane(&userdata.ControlPlaneJoinInput{
				BaseConfigure:    baseConfigure,
				CACert:           string(data.CAKeyPair.Cert),
				CAKey:            string(data.CAKeyPair.Key),
				CACertHash:       caCertHash,
				EtcdCACert:       string(data.EtcdCAKeyPair.Cert),
				EtcdCAKey:        string(data.EtcdCAKeyPair.Key),
				FrontProxyCACert: string(data.FrontProxyCAKeyPair.Cert),
				FrontProxyCAKey:  string(data.FrontProxyCAKeyPair.Key),
				SaCert:           string(data.SAKeyPair.Cert),
				SaKey:            string(data.SAKeyPair.Key),
				BootstrapToken:   data.BootstrapToken,
				ELBAddress:       data.ELBAddress,
				PrivateIP:        data.PrivateIP,
			})
			if err != nil {
				return "", err
			}
		} else {
			log.Infof("Machine %q is the first controlplane machine for cluster %q", machine.Name, cluster.Name)
			if !data.CAKeyPair.HasCertAndKey() {
				return "", fmt.Errorf("failed to run controlplane, missing CAPrivateKey")
			}

			userData, err = userdata.NewControlPlane(&userdata.ControlPlaneInput{
				BaseConfigure:     baseConfigure,
				CACert:            string(data.CAKeyPair.Cert),
				CAKey:             string(data.CAKeyPair.Key),
				EtcdCACert:        string(data.EtcdCAKeyPair.Cert),
				EtcdCAKey:         string(data.EtcdCAKeyPair.Key),
				FrontProxyCACert:  string(data.FrontProxyCAKeyPair.Cert),
				FrontProxyCAKey:   string(data.FrontProxyCAKeyPair.Key),
				SaCert:            string(data.SAKeyPair.Cert),
				SaKey:             string(data.SAKeyPair.Key),
				ELBAddress:        data.ELBAddress,
				PrivateIP:         data.PrivateIP,
				ClusterName:       cluster.Name,
				PodSubnet:         cluster.PodCidr,
				ServiceSubnet:     cluster.ServiceCidr,
				ServiceDomain:     cluster.ServiceDomain,
				KubernetesVersion: cluster.Version,
			})
			if err != nil {
				return "", err
			}
		}
	case api.RoleTypeNode:
		userData, err = userdata.NewNode(&userdata.NodeInput{
			BaseConfigure:  baseConfigure,
			CACertHash:     caCertHash,
			BootstrapToken: data.BootstrapToken,
			ELBAddress:     data.ELBAddress,
		})
		if err != nil {
			return "", err
		}
	}
	return userData, nil
}

func (d *sClusterAPIBaseDriver) ValidateDeleteCondition(ctx context.Context, userCred mcclient.TokenCredential, cluster *models.SCluster, machine *models.SMachine) error {
	return cluster.GetDriver().ValidateDeleteMachines(ctx, userCred, cluster, []manager.IMachine{machine})
}

func (d *sClusterAPIBaseDriver) SyncNetworkAddress(ctx context.Context, s *mcclient.ClientSession, machine *models.SMachine) error {
	return errors.Errorf("SyncNetworkAddress of machine %s not support", machine.GetName())
}
