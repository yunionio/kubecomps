package main

import (
	"flag"

	"yunion.io/x/log"

	"yunion.io/x/kubecomps/pkg/calico-node-agent/serve"
)

var (
	conf     string
	logLevel string
)

func init() {
	flag.StringVar(&conf, "conf", "", "config file")
	flag.StringVar(&logLevel, "log-level", "info", "log level")
	flag.Parse()
}

func main() {
	if err := log.SetLogLevelByString(log.Logger(), logLevel); err != nil {
		log.Fatalf("Set log level error: %v", logLevel)
	}
	serve.Run(conf)
}
