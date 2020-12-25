package clusters

import (
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
)

func registerClusterDriver(drv models.IClusterDriver) {
	models.RegisterClusterDriver(drv)
}
