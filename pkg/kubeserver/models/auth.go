package models

import (
	"context"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/options"
)

const (
	KUBE_SERVER_SERVICE    = "k8s"
	INTERNAL_ENDPOINT_TYPE = "internalURL"
)

func GetAdminSession() (*mcclient.ClientSession, error) {
	session := auth.AdminSession(context.TODO(), options.Options.Region, "", "", "v2")
	if session == nil {
		return nil, errors.Error("Can't get cloud session, maybe not init auth package ???")
	}
	return session, nil
}

func GetAdminCred() mcclient.TokenCredential {
	return auth.AdminCredential()
}

func GetUserSession(ctx context.Context, userCred mcclient.TokenCredential) (*mcclient.ClientSession, error) {
	s := auth.GetSession(ctx, userCred, options.Options.Region, "v2")
	if s == nil {
		return nil, errors.Errorf("Get user session nil")
	}
	return s, nil
}
