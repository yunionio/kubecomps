package client

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	cloudmod "yunion.io/x/onecloud/pkg/mcclient/modules"
)

func GetCloudSSHPrivateKey(session *mcclient.ClientSession) (string, error) {
	query := jsonutils.NewDict()
	query.Add(jsonutils.JSONTrue, "admin")
	ret, err := cloudmod.Sshkeypairs.List(session, query)
	if err != nil {
		return "", fmt.Errorf("Get admin keypair: %v", err)
	}
	if len(ret.Data) == 0 {
		return "", fmt.Errorf("Not found admin ssh keypair: %v", err)
	}
	keys := ret.Data[0]
	privateKey, err := keys.GetString("private_key")
	if err != nil {
		return "", fmt.Errorf("Get private_key: %v", err)
	}
	return privateKey, err
}

func GetKubernetesImage(session *mcclient.ClientSession) (string, error) {
	query := jsonutils.NewDict()
	query.Add(jsonutils.JSONTrue, "admin")
	ret, err := cloudmod.Images.List(session, query)
	if err != nil {
		return "", fmt.Errorf("Get kubernetes image: %v", err)
	}
	if len(ret.Data) == 0 {
		return "", fmt.Errorf("Not found kubernetes images")
	}
	image := ret.Data[0]
	return image.GetString("id")
}

func GetImage(session *mcclient.ClientSession, name string) (jsonutils.JSONObject, error) {
	return cloudmod.Images.Get(session, name, nil)
}

func GetPublicCloudImage(s *mcclient.ClientSession, input map[string]interface{}) (*jsonutils.JSONDict, error) {
	params := jsonutils.Marshal(input)
	ret, err := cloudmod.Cachedimages.List(s, params)
	if err != nil {
		return nil, errors.Wrapf(err, "list public cloud cached images %s", params)
	}
	if len(ret.Data) == 0 {
		return nil, httperrors.NewNotFoundError("not found images by params %s", params)
	}

	firstObj := ret.Data[0]
	/*
	 * details := new(api.SCachedimage)
	 * if err := firstObj.Unmarshal(details); err != nil {
	 *     return nil, err
	 * }
	 * return details, nil
	 */
	return firstObj.(*jsonutils.JSONDict), nil
}
