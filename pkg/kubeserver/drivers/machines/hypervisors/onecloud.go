package hypervisors

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/machines"
	onecloudcli "yunion.io/x/kubecomps/pkg/utils/onecloud/client"
)

func init() {
	registerDriver(newOnecloud())
}

func newOnecloud() machines.IYunionVmHypervisor {
	return new(onecloud)
}

type onecloud struct{}

func (_ onecloud) GetHypervisor() api.ProviderType {
	return api.ProviderTypeOnecloud
}

func (_ onecloud) FindSystemDiskImage(s *mcclient.ClientSession, zoneId string) (jsonutils.JSONObject, error) {
	return onecloudcli.GetImage(s, "CentOS-7.6.1810-20190430.qcow2")
}
