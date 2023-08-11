package awscli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	onecloudcli "yunion.io/x/kubecomps/pkg/utils/onecloud/client"
)

type AwsCliRC struct {
	AccessKey string
	SecretKey string
	Region    string
}

func (ac AwsCliRC) ToEnv() []string {
	return []string{
		fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", ac.AccessKey),
		fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", ac.SecretKey),
		fmt.Sprintf("AWS_DEFAULT_REGION=%s", ac.Region),
	}
}

func GetCloudProviderRC(s *mcclient.ClientSession, id string) (*jsonutils.JSONDict, error) {
	helper := onecloudcli.NewCloudproviderHelper(s)
	return helper.GetCliRC(id)
}

func GetAwsCliRCByProvider(s *mcclient.ClientSession, managerId string, regionExId string) (*AwsCliRC, error) {
	clirc, err := GetCloudProviderRC(s, managerId)
	if err != nil {
		return nil, errors.Wrapf(err, "get cloudprovider %s clirc", managerId)
	}
	ak, err := clirc.GetString("AWS_ACCESS_KEY")
	if err != nil {
		return nil, errors.Wrapf(err, "get AWS_ACCESS_KEY from cloudprovider %s", managerId)
	}
	sk, err := clirc.GetString("AWS_SECRET")
	if err != nil {
		return nil, errors.Wrapf(err, "get AWS_SECRET from cloudprovider %s", managerId)
	}
	regionParts := strings.Split(regionExId, "/")
	if len(regionParts) != 2 {
		return nil, errors.Wrapf(err, "invalid external regionId %s", regionExId)
	}
	return &AwsCliRC{
		AccessKey: ak,
		SecretKey: sk,
		Region:    regionParts[1],
	}, nil
}

func (cli *AwsCliRC) getCmd(args ...string) *exec.Cmd {
	newArgs := []string{"--region", cli.Region}
	newArgs = append(newArgs, args...)
	return exec.Command("/usr/bin/aws", newArgs...)
}

func (cli *AwsCliRC) execute(args ...string) error {
	cmd := cli.getCmd(args...)
	env := os.Environ()
	env = append(env, cli.ToEnv()...)
	cmd.Env = env

	log.Debugf("run cmd: %s, with env: %v", cmd.String(), cmd.Env)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "output: %s", output)
	}
	return nil
}

func (cli *AwsCliRC) IAMCreateRole(roleName string, policyContent string) error {
	fn := fmt.Sprintf("/tmp/aws-%s-policy.json", roleName)
	if err := os.WriteFile(fn, []byte(policyContent), 0644); err != nil {
		return errors.Wrapf(err, "write policy file: %s", fn)
	}
	return cli.execute("iam", "create-role", "--role-name", roleName, "--assume-role-policy-document", fmt.Sprintf("file://%s", fn))
}

func (cli *AwsCliRC) IAMPutRolePolicy(roleName string, policyName string, policyContent string) error {
	fn := fmt.Sprintf("/tmp/aws-%s-%s-policy.json", roleName, policyName)
	if err := os.WriteFile(fn, []byte(policyContent), 0644); err != nil {
		return errors.Wrapf(err, "write policy file: %s", fn)
	}
	return cli.execute("iam", "put-role-policy", "--role-name", roleName, "--policy-name", policyName, "--policy-document", fmt.Sprintf("file://%s", fn))
}

func (cli *AwsCliRC) IAMCreateInstanceProfile(profileName string) error {
	return cli.execute("iam", "create-instance-profile", "--instance-profile-name", profileName)
}

func (cli *AwsCliRC) IAMAddRoleToInstanceProfile(profileName string, roleName string) error {
	return cli.execute("iam", "add-role-to-instance-profile", "--instance-profile-name", profileName, "--role-name", roleName)
}

func (cli *AwsCliRC) EC2AssociateIAMInstanceProfile(instanceId string, profileName string) error {
	return cli.execute("ec2", "associate-iam-instance-profile", "--instance-id", instanceId, "--iam-instance-profile", fmt.Sprintf("Name='%s'", profileName))
}

type Tag struct {
	Key   string
	Value string
}

func NewTag(key string, value string) *Tag {
	return &Tag{
		Key:   key,
		Value: value,
	}
}

type ClusterTagValue string

const (
	ClusterTagValueShared = "shared"
	ClusterTagValueOwned  = "owned"
)

const (
	ClusterPrefixKey = "kubernetes.io/cluster"
)

func NewClusterTag(clusterName string, cv ClusterTagValue) *Tag {
	return NewTag(fmt.Sprintf("%s/%s", ClusterPrefixKey, clusterName), string(cv))
}

func (t Tag) ToArgs() string {
	return fmt.Sprintf("Key='%s',Value='%s'", t.Key, t.Value)
}

func (cli *AwsCliRC) EC2CreateTags(resource string, tag *Tag) error {
	return cli.execute("ec2", "create-tags", "--resources", resource, "--tags", tag.ToArgs())
}
