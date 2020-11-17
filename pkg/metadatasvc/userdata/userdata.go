package userdata

import (
	"strings"

	"yunion.io/x/pkg/errors"
)

const (
	ErrNotACloudConfigFile = errors.Error("userdata: not a cloud-config file")
	ErrNotAScript          = errors.Error("userdata: not a user-data script file")
)

// Map contains the user-data attributes given by the provider
type Map map[string]string

// IsScript checks whether the given content belongs to a script file
func IsScript(content string) error {
	if !strings.HasPrefix(content, "#! ") {
		return ErrNotAScript
	}

	return nil
}

// Scripts return a new amp containing only the contents
// that are valid scripts
func (m Map) Scripts() map[string]string {
	scripts := make(map[string]string)

	for k, v := range m {
		if err := IsScript(v); err == nil {
			scripts[k] = v
		}
	}

	return scripts
}

// IsCloudConfig checks whether the given content belongs to a cloud-config file
func IsCloudConfig(content string) error {
	if !strings.HasPrefix(content, "#cloud-config\n") {
		return ErrNotACloudConfigFile
	}

	return nil
}

func (m Map) CloudConfigs() map[string]string {
	confs := make(map[string]string)

	for k, v := range m {
		if err := IsCloudConfig(v); err == nil {
			confs[k] = v
		}
	}

	return confs
}
