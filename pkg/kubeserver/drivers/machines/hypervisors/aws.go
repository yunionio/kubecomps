package hypervisors

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/machines"
)

func init() {
	registerDriver(newAws())
}

func newAws() machines.IYunionVmHypervisor {
	return new(aws)
}

type aws struct{}

func (_ aws) GetHypervisor() api.ProviderType {
	return api.ProviderTypeAws
}

func (_ aws) FindSystemDiskImage(s *mcclient.ClientSession, zoneId string) (jsonutils.JSONObject, error) {
	return findSystemDiskImage(s, zoneId, func(params map[string]interface{}) map[string]interface{} {
		params["search"] = "amzn2-ami-hvm-2.0.20210126.0-x86_64-gp2"
		return params
	})
}
