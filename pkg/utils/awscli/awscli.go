package awscli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

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
	output, err := cli.executeWithOutput(args...)
	if err != nil {
		return errors.Wrapf(err, "output: %s", output)
	}
	return nil
}

func (cli *AwsCliRC) executeWithOutput(args ...string) ([]byte, error) {
	cmd := cli.getCmd(args...)
	env := os.Environ()
	env = append(env, cli.ToEnv()...)
	cmd.Env = env

	log.Debugf("run cmd: %s, with env: %v", cmd.String(), cmd.Env)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrapf(err, "output: %s\ncmd: %s", output, cmd.String())
	}
	return output, nil
}

func (cli *AwsCliRC) executeWithJSON(args ...string) (jsonutils.JSONObject, error) {
	nArgs := []string{"--output", "json"}
	nArgs = append(nArgs, args...)
	output, err := cli.executeWithOutput(nArgs...)
	if err != nil {
		return nil, errors.Wrapf(err, "output: %s", output)
	}
	obj, err := jsonutils.Parse(output)
	if err != nil {
		return nil, errors.Wrapf(err, "jsonutils.Parse with %s", output)
	}
	return obj, nil
}

func (cli *AwsCliRC) IAMCreateRole(roleName string, policyContent string) error {
	fn := fmt.Sprintf("/tmp/aws-%s-policy.json", roleName)
	if err := os.WriteFile(fn, []byte(policyContent), 0644); err != nil {
		return errors.Wrapf(err, "write policy file: %s", fn)
	}
	return cli.execute("iam", "create-role", "--role-name", roleName, "--assume-role-policy-document", fmt.Sprintf("file://%s", fn))
}

type Policy struct {
	PolicyName                    string    `json:"PolicyName"`
	PolicyId                      string    `json:"PolicyId"`
	Arn                           string    `json:"Arn"`
	Path                          string    `json:"Path"`
	DefaultVersionId              string    `json:"DefaultVersionId"`
	AttachmentCount               int       `json:"AttachmentCount"`
	PermissionsBoundaryUsageCount int       `json:"PermissionsBoundaryUsageCount"`
	IsAttachable                  bool      `json:"IsAttachable"`
	CreateDate                    time.Time `json:"CreateDate"`
	UpdateDate                    time.Time `json:"UpdateDate"`
}

func (cli *AwsCliRC) IAMListPolicies(policyName string, pathPrefix string) ([]Policy, error) {
	args := []string{
		"iam", "list-policies",
	}
	if policyName != "" {
		args = append(args, []string{"--query", fmt.Sprintf("Policies[?PolicyName==`%s`]", policyName)}...)
	}
	if pathPrefix != "" {
		args = append(args, []string{"--path-prefix", pathPrefix}...)
	}
	out, err := cli.executeWithJSON(args...)
	if err != nil {
		return nil, err
	}
	ret := []Policy{}
	if err := out.Unmarshal(&ret); err != nil {
		return nil, errors.Wrapf(err, "Unmarshal %s to []Policy", out.String())
	}
	return ret, nil
}

func (cli *AwsCliRC) IAMCreatePolicy(policyName string, policyContent string, pathName string) (*Policy, error) {
	fn := fmt.Sprintf("/tmp/aws-%s-policy.json", policyName)
	if err := os.WriteFile(fn, []byte(policyContent), 0644); err != nil {
		return nil, errors.Wrapf(err, "write policy file: %s", fn)
	}

	args := []string{
		"iam", "create-policy", "--policy-name", policyName, "--policy-document", fmt.Sprintf("file://%s", fn),
	}
	if len(pathName) != 0 {
		args = append(args, "--path", pathName)
	}
	obj, err := cli.executeWithJSON(args...)
	if err != nil {
		return nil, err
	}
	policy := new(Policy)
	if err := obj.Unmarshal(policy, "Policy"); err != nil {
		return nil, errors.Wrapf(err, "Unmarshal %s to Policy", obj.String())
	}
	return policy, nil
}

func (cli *AwsCliRC) IAMEnsurePolicy(policyName string, policyContent string, pathName string) (*Policy, error) {
	policies, err := cli.IAMListPolicies(policyName, pathName)
	if err != nil {
		return nil, errors.Wrapf(err, "IAMListPolicies")
	}
	if len(policies) != 0 {
		return &policies[0], nil
	}
	// create policy
	return cli.IAMCreatePolicy(policyName, policyContent, pathName)
}

func (cli *AwsCliRC) IAMAttachRolePolicy(roleName string, policyArn string) error {
	return cli.execute("iam", "attach-role-policy", "--role-name", roleName, "--policy-arn", policyArn)
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
	f := func() error {
		return cli.execute("ec2", "associate-iam-instance-profile", "--instance-id", instanceId, "--iam-instance-profile", fmt.Sprintf("Name='%s'", profileName))
	}

	var err error
	for i := 0; i < 2; i++ {
		err = f()
		if err == nil {
			return nil
		}
		// 关联第一次创建的 InstanceProfile 要等一段时间才生效
		time.Sleep(15 * time.Second)
	}
	return err
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
	SubnetRoleELBKey = "kubernetes.io/role/elb"
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
