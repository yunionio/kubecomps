package client

import (
	"fmt"
	"net/http"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/wait"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

func ReasonForError(err error) int {
	switch t := err.(type) {
	case *httputils.JSONClientError:
		return t.Code
	}
	return -1
}

func IsNotFound(err error) bool {
	if ReasonForError(err) == http.StatusNotFound {
		return true
	}
	return false
}

type ResourceHelper struct {
	modulebase.Manager
	session *mcclient.ClientSession
}

func NewResourceHelper(s *mcclient.ClientSession, manager modulebase.Manager) *ResourceHelper {
	return &ResourceHelper{
		Manager: manager,
		session: s,
	}
}

func (h *ResourceHelper) GetDetails(id string, output interface{}) error {
	ret, err := h.Manager.Get(h.session, id, nil)
	if err != nil {
		return errors.Wrapf(err, "get %s by id %q", h.Manager.GetKeyword(), id)
	}
	if err := ret.Unmarshal(output); err != nil {
		return errors.Wrap(err, "unmarshal json")
	}
	return nil
}

func (h *ResourceHelper) ObjectIsExists(id string) (jsonutils.JSONObject, error) {
	ret, err := h.Manager.Get(h.session, id, nil)
	if IsNotFound(err) {
		return nil, nil
	}
	return ret, err
}

func (h *ResourceHelper) indexKey(id string) string {
	return fmt.Sprintf("%s: %s", h.Manager.GetKeyword(), id)
}

func (h *ResourceHelper) WaitObjectCondition(
	id string,
	doneF func(obj jsonutils.JSONObject) (bool, error),
) error {
	interval := 5 * time.Second
	timeout := 10 * time.Minute
	return wait.Poll(interval, timeout, func() (bool, error) {
		obj, err := h.ObjectIsExists(id)
		if err != nil {
			return false, err
		}
		return doneF(obj)
	})
}

func (h *ResourceHelper) WaitObjectStatus(
	id string,
	expectStatus string,
	continueWait func(status string) bool) error {
	return h.WaitObjectCondition(
		id,
		func(obj jsonutils.JSONObject) (bool, error) {
			kw := h.indexKey(id)
			if obj == nil {
				return false, fmt.Errorf("Object %s not exists", kw)
			}
			name, _ := obj.GetString("name")
			if name != "" {
				kw = kw + "/" + name
			}
			status, _ := obj.GetString("status")
			if status == "" {
				return false, fmt.Errorf("Object %s no status", obj.PrettyString())
			}
			if status == expectStatus {
				return true, nil
			}
			if continueWait != nil && continueWait(status) {
				log.Infof("Object %s status is %q, continue waiting...", kw, status)
				return false, nil
			}
			return false, fmt.Errorf("Object %s status is %q, can't wait", kw, status)
		})
}

func (h *ResourceHelper) WaitObjectDelete(id string, continueWait func(status string) bool) error {
	return h.WaitObjectCondition(
		id,
		func(obj jsonutils.JSONObject) (bool, error) {
			if obj == nil {
				return true, nil
			}
			status, _ := obj.GetString("status")
			if status == "" {
				return false, fmt.Errorf("Object %s no status", obj.PrettyString())
			}
			kw := h.indexKey(id)
			if !continueWait(status) {
				return false, fmt.Errorf("Object %s status is %q, cancel wait", kw, status)
			}
			return false, nil
		},
	)
}
