package api

type ConfigMap struct {
	ObjectMeta
	TypeMeta
}

type ConfigMapDetail struct {
	ConfigMap

	// Data contains the configuration data.
	// Each key must be a valid DNS_SUBDOMAIN with an optional leading dot.
	Data map[string]string `json:"data,omitempty"`

	// Pods use configmap
	Pods []*Pod `json:"pods,omitempty"`
}

type ConfigMapCreateInput struct {
	NamespaceResourceCreateInput
	// required: true
	// Data contains the configuration data.
	// Each key must be a valid DNS_SUBDOMAIN with an optional leading dot.
	Data map[string]string `json:"data,omitempty"`
}

type ConfigMapUpdateInput struct {
	// required: true
	// Data contains the configuration data.
	// Each key must be a valid DNS_SUBDOMAIN with an optional leading dot.
	Data map[string]string `json:"data,omitempty"`
}
