package app

import (
	_ "yunion.io/x/kubecomps/pkg/kubeserver/drivers/machines"
	_ "yunion.io/x/kubecomps/pkg/kubeserver/drivers/machines/hypervisors"
	_ "yunion.io/x/kubecomps/pkg/kubeserver/models/drivers/release"
	_ "yunion.io/x/kubecomps/pkg/kubeserver/models/drivers/secret"
	_ "yunion.io/x/kubecomps/pkg/kubeserver/models/drivers/storageclass"
)
