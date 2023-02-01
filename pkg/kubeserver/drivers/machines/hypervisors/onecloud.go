package hypervisors

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/machines"
	onecloudcli "yunion.io/x/kubecomps/pkg/utils/onecloud/client"
)

func init() {
	registerDriver(newOnecloud())
	registerDriver(newOnecloudKvm())
}

func newOnecloud() machines.IYunionVmHypervisor {
	return new(onecloud)
}

type onecloud struct{}

func (_ onecloud) GetHypervisor() api.ProviderType {
	return api.ProviderTypeOnecloud
}

func (_ onecloud) FindSystemDiskImage(s *mcclient.ClientSession, zoneId string) (jsonutils.JSONObject, error) {
	preferImg, err := onecloudcli.GetImage(s, "CentOS-7.6.1810-20190430.qcow2")
	if err == nil {
		return preferImg, nil
	}
	params := map[string]interface{}{
		"distributions":              []string{"CentOS", "Kylin"},
		"distribution_precise_match": true,
		"scope":                      "system",
		"filter.0":                   "format.notequals(iso)",
	}
	ret, err := onecloudcli.ListImages(s, jsonutils.Marshal(params).(*jsonutils.JSONDict))
	if err != nil {
		return nil, errors.Wrap(err, "list images")
	}
	if ret.Total == 0 {
		return nil, httperrors.NewNotFoundError("Not found CentOS or Kylin image")
	}
	return ret.Data[0], nil
}

func newOnecloudKvm() machines.IYunionVmHypervisor {
	return new(onecloudKvm)
}

type onecloudKvm struct {
	onecloud
}

func (_ onecloudKvm) GetHypervisor() api.ProviderType {
	return api.ProviderTypeOnecloudKvm
}
