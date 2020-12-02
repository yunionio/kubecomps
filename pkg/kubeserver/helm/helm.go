package helm

import (
	"os"

	"helm.sh/helm/v3/pkg/helmpath/xdg"
)

func InitEnv(dataDir string) {
	os.Setenv(xdg.CacheHomeEnvVar, dataDir)
	os.Setenv(xdg.ConfigHomeEnvVar, dataDir)
	os.Setenv(xdg.DataHomeEnvVar, dataDir)
}
