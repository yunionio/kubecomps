package secret

import (
	"yunion.io/x/onecloud/pkg/httperrors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
)

func init() {
	models.GetSecretManager().RegisterDriver(
		api.SecretTypeCephCSI,
		newCephCSI(),
	)
}

type cephCSI struct{}

func (c cephCSI) ValidateCreateData(input *api.SecretCreateInput) error {
	conf := input.CephCSI
	if conf == nil {
		return httperrors.NewInputParameterError("ceph csi config is empty")
	}
	if conf.UserId == "" {
		return httperrors.NewInputParameterError("userId is empty")
	}
	if conf.UserKey == "" {
		return httperrors.NewInputParameterError("userKey is empty")
	}
	return nil
}

func (c cephCSI) ToData(input *api.SecretCreateInput) (map[string]string, error) {
	conf := input.CephCSI
	ret := map[string]string{
		"userID":  conf.UserId,
		"userKey": conf.UserKey,
	}
	if conf.EncryptionPassphrase != "" {
		ret["encryptionPassphrase"] = conf.EncryptionPassphrase
	}
	return ret, nil
}

func newCephCSI() models.ISecretDriver {
	return new(cephCSI)
}
