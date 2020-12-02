package k8s

import (
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func GetConfig(kubeConfigFile string) (*rest.Config, error) {
	return clientcmd.BuildConfigFromFlags("", kubeConfigFile)
}
