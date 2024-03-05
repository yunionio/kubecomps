package hypervisors

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	ocapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/machines"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	"yunion.io/x/kubecomps/pkg/utils/awscli"
	onecloudcli "yunion.io/x/kubecomps/pkg/utils/onecloud/client"
)

func init() {
	registerDriver(newAws())
}

func newAws() machines.IYunionVmHypervisor {
	return aws{
		newBaseHypervisor(api.ProviderTypeAws),
	}
}

type aws struct {
	*baseHypervisor
}

func (_ aws) FindSystemDiskImage(s *mcclient.ClientSession, zoneId string) (jsonutils.JSONObject, error) {
	return findSystemDiskImage(s, zoneId, func(params map[string]interface{}) map[string]interface{} {
		params["filter.0"] = "name.contains(ubuntu/images/hvm-ssd/ubuntu-focal-20.04-amd64-server)"
		// params["filter.1"] = "name.contains(ubuntu-minimal/images/hvm-ssd/ubuntu-focal-20.04-amd64-minimal)"
		params["filter_any"] = true
		// params["search"] = "amzn2-ami-hvm-2.0.20230320.0-x86_64-ebs"
		return params
	})
}

func (_ aws) PostPrepareServerResource(ctx context.Context, s *mcclient.ClientSession, m *models.SMachine, srv *ocapi.ServerDetails) error {
	cls, err := m.GetCluster()
	if err != nil {
		return errors.Wrapf(err, "get machine %s cluster", m.GetName())
	}
	vpcId := cls.VpcId
	if vpcId == "" {
		return errors.Wrapf(err, "vpcId is empty of cluster %s", cls.GetName())
	}
	vpc, err := onecloudcli.NewVpcHelper(s).GetDetails(vpcId)
	if err != nil {
		return errors.Wrapf(err, "get vpc %s details from region", vpcId)
	}

	netIds := []string{}
	for _, nic := range srv.Nics {
		if nic.VpcId == vpcId {
			netIds = append(netIds, nic.NetworkId)
			break
		}
	}
	if len(netIds) == 0 {
		return errors.Wrapf(err, "not found subnetwork id of vpc %s", vpcId)
	}

	cloudRegion, err := onecloudcli.NewCloudregionHelper(s).GetDetails(vpc.CloudregionId)
	if err != nil {
		return errors.Wrapf(err, "get cloudregion %s details from region", vpc.CloudregionId)
	}
	managerId := vpc.ManagerId
	clirc, err := awscli.GetAwsCliRCByProvider(s, managerId, cloudRegion.ExternalId)
	if err != nil {
		return errors.Wrapf(err, "get cloudprovider %s clirc", managerId)
	}
	// ref: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/iam-roles-for-amazon-ec2.html#attach-iam-role
	// 1.
	roleName := "kubeserver_node_access"
	if err := clirc.IAMCreateRole(roleName, EC2_ROLE_TRUST_POLICY); err != nil {
		if !strings.Contains(err.Error(), "already exists.") {
			return errors.Wrap(err, "IAMCreateRole")
		}
	}

	pathName := "/kubeserver/"
	// 2.
	for key, content := range map[string]string{
		"cni":      EC2_VPC_CNI_POLICY,
		"provider": EC2_CLOUD_PROVIDER_POLICY,
		"lb":       EC2_LB_CONTROLLER_POLICY,
		"csi":      EC2_EBS_CSI_POLICY,
	} {
		policyName := fmt.Sprintf("kubeserver_%s", key)
		policyObj, err := clirc.IAMEnsurePolicy(policyName, content, pathName)
		if err != nil {
			return errors.Wrapf(err, "IAMEnsurePolicy %s", policyName)
		}
		if err := clirc.IAMAttachRolePolicy(roleName, policyObj.Arn); err != nil {
			log.Warningf("IAMAttachRolePolicy: %v", err)
			if !strings.Contains(err.Error(), "already ") {
				return errors.Wrap(err, "IAMAttachRolePolicy")
			}
		}
	}
	// 3.
	instanceProfile := fmt.Sprintf("%s_profile", roleName)
	if err := clirc.IAMCreateInstanceProfile(instanceProfile); err != nil {
		log.Warningf("IAMCreateInstanceProfile: %v", err)
		if !strings.Contains(err.Error(), "already exists") {
			return errors.Wrap(err, "IAMCreateInstanceProfile")
		}
	}
	// 4.
	if err := clirc.IAMAddRoleToInstanceProfile(instanceProfile, roleName); err != nil {
		log.Warningf("IAMAddRoleToInstanceProfile: %v", err)
		if !strings.Contains(err.Error(), "Cannot exceed quota for InstanceSessionsPerInstanceProfile") {
			return errors.Wrap(err, "IAMAddRoleToInstanceProfile")
		}
	}
	instanceId := srv.ExternalId
	if instanceId == "" {
		return errors.Wrapf(err, "server %s external_id is empty", srv.Name)
	}
	// 5.
	if err := clirc.EC2AssociateIAMInstanceProfile(instanceId, instanceProfile); err != nil {
		log.Warningf("EC2AssociateIAMInstanceProfile: %v", err)
		if !strings.Contains(err.Error(), "There is an existing association for instance") {
			return errors.Wrap(err, "EC2AssociateIAMInstanceProfile")
		}
	}

	// 6. tag instance to cluster
	if err := clirc.EC2CreateTags(instanceId, awscli.NewClusterTag(cls.GetName(), awscli.ClusterTagValueOwned)); err != nil {
		return errors.Wrap(err, "EC2CreateTags for instance")
	}

	// 7. tag subnet to cluster
	for _, netId := range netIds {
		details, err := onecloudcli.NewNetworkHelper(s).GetDetails(netId)
		if err != nil {
			return errors.Wrapf(err, "get network %s details", netId)
		}
		if err := clirc.EC2CreateTags(details.ExternalId, awscli.NewClusterTag(cls.GetName(), awscli.ClusterTagValueShared)); err != nil {
			return errors.Wrapf(err, "EC2CreateTags for subnet %s", details.Name)
		}

		if err := clirc.EC2CreateTags(details.ExternalId, awscli.NewTag(awscli.SubnetRoleELBKey, "1")); err != nil {
			return errors.Wrapf(err, "EC2CreateTags %s for subnet %s", awscli.SubnetRoleELBKey, details.Name)
		}
	}

	log.Infof("cluster %s manager id is %s", cls.GetName(), managerId)
	return nil
}
