package config

import (
	"io/ioutil"
	"os"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/calico-node-agent/types"
)

func GetNodeConfigByFile(cf string) (*types.NodeConfig, error) {
	content, err := ioutil.ReadFile(cf)
	if err != nil {
		return nil, errors.Wrapf(err, "read file %s", cf)
	}
	return GetNodeConfigByBytes(string(content))
}

func GetNodeConfigByBytes(content string) (*types.NodeConfig, error) {
	obj, err := jsonutils.ParseYAML(content)
	if err != nil {
		return nil, errors.Wrap(err, "parse config file yaml contents")
	}
	conf := new(types.NodeConfig)
	if err := obj.Unmarshal(conf); err != nil {
		return nil, errors.Wrap(err, "unmarshal node config")
	}
	fillNodeConfig(conf)
	if err := conf.Validate(); err != nil {
		return nil, errors.Wrap(err, "validate node config")
	}
	return conf, nil
}

func fillNodeConfig(conf *types.NodeConfig) {
	if conf.NodeName == "" {
		val, ok := os.LookupEnv(types.EnvKeyNodeName)
		if ok {
			conf.NodeName = val
		}
	}
}
