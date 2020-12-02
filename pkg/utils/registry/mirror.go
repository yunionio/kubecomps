package registry

import (
	"fmt"
)

func MirrorImage(imageRepo, name string, tag string, prefix string) string {
	if tag == "" {
		tag = "latest"
	}
	name = fmt.Sprintf("%s:%s", name, tag)
	if prefix != "" {
		name = fmt.Sprintf("%s-%s", prefix, name)
	}
	return fmt.Sprintf("%s/%s", imageRepo, name)
}
