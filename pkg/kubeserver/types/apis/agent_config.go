package apis

type HostRegisterConfig struct {
	AgentConfig   AgentConfig   `json:"agentConfig"`
	DockerdConfig DockerdConfig `json:"dockerdConfig"`
}

type AgentConfig struct {
	ServerUrl string `json:"serverUrl"`
	Token     string `json:"token"`
	ClusterId string `json:"clusterId"`
	NodeId    string `json:"nodeId"`
}

type DockerdConfig struct {
	LiveRestore        bool     `json:"live-restore"`
	RegistryMirrors    []string `json:"registry-mirrors"`
	InsecureRegistries []string `json:"insecure-registries"`
	Graph              string   `json:"graph"`
	Bip                string   `json:"bip,omitempty"`
}
