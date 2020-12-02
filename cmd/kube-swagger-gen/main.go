package main

import (
	"os"
	"path/filepath"

	"k8s.io/gengo/args"
	"k8s.io/klog"

	cgenerators "yunion.io/x/code-generator/pkg/swagger-gen/generators"
	"yunion.io/x/kubecomps/pkg/kube-swagger-gen/generators"
)

func main() {
	klog.InitFlags(nil)
	arguments := args.Default()

	// Override defaults.
	arguments.OutputFileBaseName = "zz_generated.swagger_spec"
	arguments.GoHeaderFilePath = filepath.Join(args.DefaultSourceTree(), "yunion.io/x/onecloud/scripts/copyright.txt")

	if err := arguments.Execute(
		cgenerators.NameSystems(),
		cgenerators.DefaultNameSystem(),
		generators.Packages,
	); err != nil {
		klog.Errorf("Error: %v", err)
		os.Exit(1)
	}
}
