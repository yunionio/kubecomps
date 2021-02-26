package hypervisors

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/machines"
	onecloudcli "yunion.io/x/kubecomps/pkg/utils/onecloud/client"
)

func registerDriver(drv machines.IYunionVmHypervisor) {
	machines.GetYunionVMDriver().RegisterHypervisor(drv)
}

func findSystemDiskImage(
	s *mcclient.ClientSession,
	zoneId string,
	getParams func(map[string]interface{}) map[string]interface{},
) (jsonutils.JSONObject, error) {
	params := map[string]interface{}{
		"order_by":   "ref_count",
		"order":      "desc",
		"valid":      true,
		"image_type": "system",
	}
	if zoneId != "" {
		params["zone_id"] = zoneId
	}
	params = getParams(params)

	return onecloudcli.GetPublicCloudImage(s, params)
}
