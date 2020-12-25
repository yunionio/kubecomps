package kubespray

import (
	"regexp"

	"yunion.io/x/pkg/errors"
)

var (
	KubernetesVersionRegexp = regexp.MustCompile(`^1\.\d{1,2}\.\d$`)
)

const (
	ErrKubernetesVersionEmpty         = errors.Error("KubernetesVersion is empty")
	ErrKubernetesVersionInvalidFormat = errors.Error("KubernetesVersion invalid format")
)

type CommonVars struct {
	DockerRegistryMirrors    []string `json:"docker_registry_mirrors,allowempty"`
	DockerInsecureRegistries []string `json:"docker_insecure_registries,allowempty"`
	KubernetesVersion        string   `json:"kubernetes_version"`
}

func ValidateKubernetesVersion(kv string) error {
	if kv == "" {
		return ErrKubernetesVersionEmpty
	}

	if match := KubernetesVersionRegexp.MatchString(kv); !match {
		return errors.Wrapf(ErrKubernetesVersionInvalidFormat, "%s", kv)
	}

	return nil
}
