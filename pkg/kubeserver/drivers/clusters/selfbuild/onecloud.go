package selfbuild

import "yunion.io/x/kubecomps/pkg/kubeserver/api"

type sOnecloudDriver struct {
	*sBaseGuestDriver
}

func NewOnecloudDriver() ISelfBuildDriver {
	return &sOnecloudDriver{
		sBaseGuestDriver: newBaseGuestDriver(api.ProviderTypeOnecloud),
	}
}

func NewOnecloudKvmDriver() ISelfBuildDriver {
	return &sOnecloudDriver{
		sBaseGuestDriver: newBaseGuestDriver(api.ProviderTypeOnecloudKvm),
	}
}
