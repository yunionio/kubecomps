package imported

type ExternalK8s struct {
	*k8sBaseDriver
}

func NewExternalK8s() *ExternalK8s {
	return &ExternalK8s{
		k8sBaseDriver: newK8sBaseDriver(),
	}
}
