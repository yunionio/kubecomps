package apis

type NodeAddOption struct {
	Cluster          string        `json:"cluster"`
	Roles            []string      `json:"roles"`
	Name             string        `json:"name,omitempty"`
	Host             string        `json:"host,omitempty"`
	HostnameOverride string        `json:"hostname_override,omitempty"`
	DockerdConfig    DockerdConfig `json:"dockerd_config,omitempty"`
}
