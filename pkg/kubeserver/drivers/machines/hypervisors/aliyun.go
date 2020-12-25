package hypervisors

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/machines"
)

func init() {
	registerDriver(newAliyun())
}

func newAliyun() machines.IYunionVmHypervisor {
	return new(aliyun)
}

type aliyun struct{}

func (_ aliyun) GetHypervisor() api.ProviderType {
	return api.ProviderTypeAliyun
}

func (_ aliyun) FindSystemDiskImage(s *mcclient.ClientSession, zoneId string) (jsonutils.JSONObject, error) {
	return findSystemDiskImage(s, zoneId, func(params map[string]interface{}) map[string]interface{} {
		params["search"] = "7.9"
		params["filter.0"] = "name.contains(CentOS)"
		return params
	})
}
