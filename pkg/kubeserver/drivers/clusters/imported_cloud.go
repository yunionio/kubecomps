package clusters

import (
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/clusters/imported"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
)

func init() {
	for _, drv := range []imported.ICloudImportDriver{
		imported.NewCloudAliyunK8s(),
		imported.NewCloudAzureK8s(),
		imported.NewCloudQcloudK8s(),
	} {
		models.RegisterClusterDriver(newCloudImportDriver(drv))
	}
}
