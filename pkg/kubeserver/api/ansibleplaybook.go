package api

type LocalObjectReference struct {
	Name string `json:"name,omitempty"`
}

type AnsiblePlaybook struct {
	ObjectTypeMeta

	PlaybookTemplateRef *LocalObjectReference `json:"playbookTemplateRef,omitempty"`

	MaxRetryTime *int32 `json:"maxRetryTimes,omitempty"`
	AnsiblePlaybookStatus
}

type AnsiblePlaybookStatus struct {
	Status       string              `json:"status"`
	ExternalInfo AnsiblePlaybookInfo `json:"externalInfo"`
	TryTimes     int32               `json:"tryTimes"`
}

type AnsiblePlaybookInfo struct {
	OnecloudExternalInfoBase
	// OUtput is ansible playbook result output
	Output string `json:"output,omitempty"`
}
