package main

import (
	"os"
	"path/filepath"

	"yunion.io/x/log"

	"yunion.io/x/kubecomps/pkg/cni-plugin/plugin"
)

func main() {
	// Use the name of the binary to determine which routine to run.
	_, filename := filepath.Split(os.Args[0])
	switch filename {
	case "ocnet-cni":
		plugin.Main("unknown version")
	default:
		log.Fatalf("Unsupported %s", filename)
	}
}
