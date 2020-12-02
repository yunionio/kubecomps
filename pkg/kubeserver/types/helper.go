package types

import (
	"strings"

	"yunion.io/x/log"
)

func isASCII(s string) bool {
	for _, c := range s {
		if c > 127 {
			return false
		}
	}
	return true
}

func ConvertProjectToNamespace(name string) string {
	trans := func(name string, olds []string, new string) string {
		for _, ch := range olds {
			name = strings.Replace(name, ch, new, -1)
		}
		return strings.ToLower(name)
	}
	if !isASCII(name) {
		log.Warningf("Project name %q is not ASCII string, skip it", name)
		return ""
	}
	validName := trans(name,
		[]string{"/", `\`, ".", "?", "!", "@", "#", "$", "%", "^", "&", "*", "(", ")", "_", "+", "="}, "-")
	log.Debugf("Do trans %q => %q", name, validName)
	return validName
}
