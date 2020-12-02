package api

const (
	DefaultDockerGraphDir        = "/opt/docker"
	DefaultDockerRegistryMirror1 = "https://lje6zxpk.mirror.aliyuncs.com"
	DefaultDockerRegistryMirror2 = "https://lms7sxqp.mirror.aliyuncs.com"
	DefaultDockerRegistryMirror3 = "https://registry.docker-cn.com"
)

type DockerConfigLogOpts struct {
	MaxSize string `json:"max-size"`
}

type DockerConfig struct {
	Graph              string              `json:"graph"`
	RegistryMirrors    []string            `json:"registry-mirrors"`
	InsecureRegistries []string            `json:"insecure-registries"`
	Bridge             string              `json:"bridge"`
	Iptables           bool                `json:"iptables"`
	LiveRestore        bool                `json:"live-restore"`
	ExecOpts           []string            `json:"exec-opts"`
	LogDriver          string              `json:"log-driver"`
	LogOpts            DockerConfigLogOpts `json:"log-opts"`
	StorageDriver      string              `json:"storage-driver,omitempty"`
	StorageOpts        []string            `json:"storage-opts,omitempty"`
}
