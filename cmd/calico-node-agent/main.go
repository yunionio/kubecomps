package main

import (
	"flag"

	"yunion.io/x/kubecomps/pkg/calico-node-agent/serve"
)

var (
	conf string
)

func init() {
	flag.StringVar(&conf, "conf", "", "config file")
	flag.Parse()
}

func main() {
	serve.Run(conf)
}
