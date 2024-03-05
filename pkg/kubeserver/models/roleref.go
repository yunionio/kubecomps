package models

import (
	"context"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
)

type SRoleRefResourceBaseManager struct{}

type SRoleRefResourceBase struct {
	Subjects *api.Subjects `list:"user" update:"user" create:"optional"`
	RoleRef  *api.RoleRef  `list:"user" update:"user" create:"required"`
}

type IRoleBaseManager interface {
	db.IModelManager
	GetRoleKind() string
}

func (m *SRoleRefResourceBaseManager) ValidateRoleRef(ctx context.Context, roleObjManager IRoleBaseManager, userCred mcclient.TokenCredential, ref *api.RoleRef) error {
	if ref == nil {
		return httperrors.NewNotEmptyError("roleRef must provided")
	}
	kind := roleObjManager.GetRoleKind()
	if ref.Kind != kind {
		return httperrors.NewNotAcceptableError("role reference kind must %s, input %s", kind, ref.Kind)
	}
	refObj, err := roleObjManager.FetchByIdOrName(ctx, userCred, ref.Name)
	if err != nil {
		return err
	}
	ref.Name = refObj.GetName()
	return nil
}

func (obj *SRoleRefResourceBase) CustomizeCreate(input *api.RoleRef) error {
	if input == nil {
		return httperrors.NewNotEmptyError("input roleRef is nil")
	}
	obj.RoleRef = input
	return nil
}
