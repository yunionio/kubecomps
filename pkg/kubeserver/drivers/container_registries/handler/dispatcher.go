package handler

import (
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

type IDispatchHandler interface {
	dispatcher.IModelDispatchHandler
}

type IModelManager interface {
	Keyword() string
	KeywordPlural() string
}

type modelHandler struct {
	manager IModelManager
}

func NewModelHandler(manager IModelManager) IDispatchHandler {
	return &modelHandler{
		manager: manager,
	}
}

func (mh *modelHandler) Filter(f appsrv.FilterHandler) appsrv.FilterHandler {
	return auth.Authenticate(f)
}

func (mh *modelHandler) Keyword() string {
	return mh.manager.Keyword()
}

func (mh *modelHandler) KeywordPlural() string {
	return mh.manager.KeywordPlural()
}
