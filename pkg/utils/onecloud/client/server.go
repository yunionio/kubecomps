package client

import (
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	cloudansible "yunion.io/x/onecloud/pkg/util/ansible"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
)

const (
	NotFoundMsg = "NotFoundError"
)

func IsNotFoundError(err error) bool {
	if httpErr, ok := err.(*httputils.JSONClientError); ok {
		if httpErr.Code == http.StatusNotFound {
			return true
		}
	}
	if strings.Contains(err.Error(), NotFoundMsg) {
		return true
	}
	return false
}

type ServerHelper struct {
	*ResourceHelper
}

func NewServerHelper(s *mcclient.ClientSession) *ServerHelper {
	return &ServerHelper{
		ResourceHelper: NewResourceHelper(s, &modules.Servers),
	}
}

func (h *ServerHelper) Servers() *modules.ServerManager {
	return h.ResourceHelper.Manager.(*modules.ServerManager)
}

func (h *ServerHelper) continueWait(status string) bool {
	if strings.HasSuffix(status, "_fail") || strings.HasSuffix(status, "_failed") {
		return false
	}
	return true
}

func (h *ServerHelper) WaitRunning(id string) error {
	return h.WaitObjectStatus(id, api.VM_RUNNING, h.continueWait)
}

func (h *ServerHelper) WaitDelete(id string) error {
	return h.WaitObjectDelete(id, h.continueWait)
}

type ServerLoginInfo struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *ServerHelper) GetLoginInfo(id string) (*ServerLoginInfo, error) {
	ret, err := h.Servers().GetLoginInfo(h.session, id, nil)
	if err != nil {
		return nil, err
	}
	info := new(ServerLoginInfo)
	if err := ret.Unmarshal(info); err != nil {
		return nil, err
	}
	if len(info.Username) == 0 || len(info.Password) == 0 {
		return nil, errors.Error("Empty username or password")
	}
	return info, nil
}

type ServerSSHLoginInfo struct {
	*ServerLoginInfo
	Hostname   string
	EIP        string
	PrivateIP  string
	PrivateKey string
}

func (info ServerSSHLoginInfo) GetAccessIP() string {
	if info.EIP != "" {
		return info.EIP
	}
	return info.PrivateIP
}

func (h *ServerHelper) GetSSHLoginInfo(id string) (*ServerSSHLoginInfo, error) {
	privateKey, err := GetCloudSSHPrivateKey(h.session)
	if err != nil {
		return nil, errors.Wrapf(err, "GetCloudSSHPrivateKey")
	}
	detail, err := h.GetDetails(id)
	if err != nil {
		return nil, errors.Wrapf(err, "Get server detail")
	}
	privateIP, err := h.GetPrivateIP(id)
	if err != nil {
		return nil, errors.Wrapf(err, "Get server %q PrivateIP", id)
	}
	loginInfo, err := h.GetLoginInfo(id)
	if err != nil {
		return nil, errors.Wrapf(err, "Get server %q loginInfo", id)
	}
	loginInfo.Password = ""
	loginInfo.Username = cloudansible.PUBLIC_CLOUD_ANSIBLE_USER
	return &ServerSSHLoginInfo{
		ServerLoginInfo: loginInfo,
		Hostname:        detail.Name,
		EIP:             detail.Eip,
		PrivateIP:       privateIP,
		PrivateKey:      privateKey,
	}, nil
}

func (h *ServerHelper) ListServerNetworks(id string) ([]*api.SGuestnetwork, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.JSONTrue, "system")
	params.Add(jsonutils.JSONTrue, "admin")
	ret, err := modules.Servernetworks.ListDescendent(h.session, id, params)
	if err != nil {
		return nil, err
	}
	if len(ret.Data) == 0 {
		return nil, errors.Errorf("Not found networks by id: %s", id)
	}
	objs := make([]*api.SGuestnetwork, 0)
	for _, data := range ret.Data {
		obj := new(api.SGuestnetwork)
		if err := data.Unmarshal(obj); err != nil {
			return nil, err
		}
		objs = append(objs, obj)
	}
	return objs, nil
}

func (h *ServerHelper) GetPrivateIP(id string) (string, error) {
	nets, err := h.ListServerNetworks(id)
	if err != nil {
		return "", errors.Wrap(err, "list server networks")
	}
	return nets[0].IpAddr, nil
}

func (h *ServerHelper) GetEIP(id string) (string, error) {
	obj, err := h.GetDetails(id)
	if err != nil {
		return "", errors.Wrap(err, "get cloud server details")
	}
	if obj.Eip == "" {
		return "", errors.Errorf("server %s not found eip", id)
	}
	return obj.Eip, nil
}

func (h *ServerHelper) ListNetworkAddress(id string) ([]*api.NetworkAddressDetails, error) {
	input := new(api.NetworkAddressListInput)
	zeroLimit := 0
	input.Limit = &zeroLimit
	input.GuestId = []string{id}
	ret, err := modules.NetworkAddresses.List(h.session, input.JSON(input))
	if err != nil {
		return nil, err
	}

	objs := make([]*api.NetworkAddressDetails, 0)
	for _, obj := range ret.Data {
		out := new(api.NetworkAddressDetails)
		if err := obj.Unmarshal(out); err != nil {
			return nil, err
		}
		objs = append(objs, out)
	}

	return objs, nil
}

func (h *ServerHelper) AttachNetworkAddress(id string, ip string) error {
	rInput := &api.NetworkAddressCreateInput{
		GuestId: id,
		// TODO: support specify network index
		// always use first network currently
		ParentType:        api.NetworkAddressParentTypeGuestnetwork,
		GuestnetworkIndex: 0,
		Type:              api.NetworkAddressTypeSubIP,
		IPAddr:            ip,
	}

	if _, err := modules.NetworkAddresses.Create(h.session, jsonutils.Marshal(rInput)); err != nil {
		return errors.Wrap(err, "Attach network address")
	}

	return nil
}

func (h *ServerHelper) GetDetails(id string) (*api.ServerDetails, error) {
	out := new(api.ServerDetails)
	if err := h.ResourceHelper.GetDetails(id, out); err != nil {
		return nil, err
	}

	return out, nil
}

type ServerEIP struct {
	IP   string
	Mode string
}

func (h *ServerHelper) CreateEIP(srv *api.ServerDetails) (*ServerEIP, error) {
	// TODO: public cloud use charge_type by traffic
	input := map[string]interface{}{
		"auto_dellocate": true,
		// "charge_type":    api.EIP_CHARGE_TYPE_BY_TRAFFIC,
		"charge_type": api.EIP_CHARGE_TYPE_BY_BANDWIDTH,
		"bandwidth":   100,
		// "bandwidth": 1,
	}

	id := srv.Id

	if !utils.IsInStringArray(srv.Hypervisor, []string{api.HYPERVISOR_KVM, api.HYPERVISOR_BAREMETAL, api.HYPERVISOR_ESXI}) {
		input["charge_type"] = api.EIP_CHARGE_TYPE_BY_TRAFFIC
		// delete(input, "bandwidth")
	}

	_, err := h.Servers().PerformAction(h.session, id, "create-eip", jsonutils.Marshal(input))
	if err != nil {
		return nil, errors.Wrapf(err, "server create eip")
	}
	if err := h.WaitRunning(id); err != nil {
		return nil, errors.Wrap(err, "wait server running after create eip")
	}
	detail, err := h.GetDetails(id)
	if err != nil {
		return nil, errors.Wrap(err, "get server detail after crate eip")
	}
	return &ServerEIP{
		IP:   detail.Eip,
		Mode: detail.EipMode,
	}, nil
}
