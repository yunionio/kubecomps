package client

import (
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/pkg/errors"
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

func (h *ServerHelper) GetDetails(id string) (*api.ServerDetails, error) {
	obj, err := h.Servers().Get(h.session, id, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "get server by id %q", id)
	}
	out := new(api.ServerDetails)
	if err := obj.Unmarshal(out); err != nil {
		return nil, errors.Wrap(err, "unmarshal json")
	}

	return out, nil
}

type ServerEIP struct {
	IP   string
	Mode string
}

func (h *ServerHelper) CreateEIP(id string) (*ServerEIP, error) {
	// TODO: public cloud use charge_type by traffic
	input := map[string]interface{}{
		"auto_dellocate": true,
		// "charge_type":    api.EIP_CHARGE_TYPE_BY_TRAFFIC,
		"charge_type": api.EIP_CHARGE_TYPE_BY_BANDWIDTH,
		"bandwidth":   100,
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
