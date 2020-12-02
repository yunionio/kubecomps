package api

type AnsiblePlaybookTemplate struct {
	ObjectTypeMeta
	AnsiblePlaybookTemplateSpec
}

type AnsiblePlaybookTemplateSpec struct {
	// Playbook describe the main content of ansible playbook which should be in yaml format
	Playbook string `json:"playbook"`
	// Requirements describe the source of roles dependent on Playbook
	Requirements string `json:"requirements"`
	// Files describe the associated file tree and file content which should be in json format
	Files string `json:"files,omitempty"`
	// Vars describe the vars to apply this ansible playbook
	Vars []AnsiblePlaybookTemplateVar `json:"vars,omitempty"`
}

type AnsiblePlaybookTemplateVar struct {
	Name string `json:"name"`
	// Required indicates whether this variable is required
	Required *bool `json:"required"`
	// Default describe the default value of this variable
	Default interface{} `json:"default,omitempty"`
}
