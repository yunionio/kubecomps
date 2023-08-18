package models

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
)

const (
	machinesDeployIdsKey       = "machineIds"
	clusterDeployActionKey     = "action"
	clusterDeploySkipDownloads = "skipDownloads"
)

func SetDataDeployMachineIds(data *jsonutils.JSONDict, ids ...string) error {
	var arrary *jsonutils.JSONArray

	if !data.Contains(machinesDeployIdsKey) {
		arrary = jsonutils.NewArray()
	} else {
		arraryObj, err := data.Get(machinesDeployIdsKey)
		if err != nil {
			return errors.Wrapf(err, "get %s from data", machinesDeployIdsKey)
		}
		arrary = arraryObj.(*jsonutils.JSONArray)
	}

	for _, id := range ids {
		arrary.Add(jsonutils.NewString(id))
	}

	data.Set(machinesDeployIdsKey, arrary)

	return nil
}

func GetDataDeployMachineIds(data *jsonutils.JSONDict) ([]string, error) {
	ids, err := data.GetArray(machinesDeployIdsKey)
	if err != nil {
		return nil, errors.Wrapf(err, "get array")
	}

	ret := make([]string, len(ids))
	for index := range ids {
		id, err := ids[index].GetString()
		if err != nil {
			return nil, errors.Wrap(err, "get id string")
		}
		ret[index] = id
	}

	return ret, nil
}

func SetDataDeployAction(data *jsonutils.JSONDict, action api.ClusterDeployAction) {
	data.Add(jsonutils.NewString(string(action)), clusterDeployActionKey)
}

func GetDataDeployAction(data *jsonutils.JSONDict) (api.ClusterDeployAction, error) {
	action, err := data.GetString(clusterDeployActionKey)
	if err != nil {
		return "", err
	}
	return api.ClusterDeployAction(action), nil
}

func SetDataDeploySkipDownloads(data *jsonutils.JSONDict, skipDownloads bool) {
	data.Add(jsonutils.NewBool(skipDownloads), clusterDeploySkipDownloads)
}

func GetDataDeploySkipDownloads(data *jsonutils.JSONDict) bool {
	skipDownloads, _ := data.Bool(clusterDeploySkipDownloads)
	return skipDownloads
}
