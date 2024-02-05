package plugin

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"yunion.io/x/pkg/errors"
)

const (
	K8S_POD_NAMESPACE          = "K8S_POD_NAMESPACE"
	K8S_POD_NAME               = "K8S_POD_NAME"
	K8S_POD_INFRA_CONTAINER_ID = "K8S_POD_INFRA_CONTAINER_ID"
	K8S_POD_UID                = "K8S_POD_UID"
)

type PodInfo struct {
	Id          string
	Name        string
	Namespace   string
	ContainerId string
}

func NewPodInfoFromCNIArgs(args string) (*PodInfo, error) {
	segs := strings.Split(args, ";")
	ret := new(PodInfo)
	for _, seg := range segs {
		kv := strings.Split(seg, "=")
		if len(kv) != 2 {
			return nil, fmt.Errorf("Invalid args part: %q", seg)
		}
		key := kv[0]
		val := kv[1]
		switch key {
		case K8S_POD_NAMESPACE:
			ret.Namespace = val
		case K8S_POD_NAME:
			ret.Name = val
		case K8S_POD_INFRA_CONTAINER_ID:
			ret.ContainerId = val
		case K8S_POD_UID:
			ret.Id = val
		}
	}
	if ret.Id == "" {
		return nil, errors.Errorf("Not found %s from args %s", K8S_POD_UID, args)
	}
	return ret, nil
}

func (p PodInfo) GetDescPath() string {
	return filepath.Join(GetCloudServerDir(), p.Id, "desc")
}

type CloudPod struct {
	*PodInfo
	desc *PodDesc
}

func GetCloudServerDir() string {
	// TODO: make it configurable
	return "/opt/cloud/workspace/servers"
}

func NewCloudPodFromCNIArgs(args string) (*CloudPod, error) {
	info, err := NewPodInfoFromCNIArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "NewPodInfoFromCNIArgs")
	}
	descFile := info.GetDescPath()
	descData, err := ioutil.ReadFile(descFile)
	if err != nil {
		return nil, errors.Wrap(err, "read desc file")
	}
	desc := new(PodDesc)
	if err := json.Unmarshal(descData, desc); err != nil {
		return nil, errors.Wrap(err, "json.Unmarshal")
	}
	pod := &CloudPod{
		PodInfo: info,
		desc:    desc,
	}
	return pod, nil
}

func (p CloudPod) GetDesc() *PodDesc {
	return p.desc
}
