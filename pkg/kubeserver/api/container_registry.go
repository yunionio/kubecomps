package api

import (
	"yunion.io/x/onecloud/pkg/apis"
)

type ContainerRegistryType string

const (
	ContainerRegistryTypeHarbor = "harbor"
)

type ContainerRegistryListInput struct {
	apis.StatusInfrasResourceBaseListInput

	Type string `json:"type"`
}

type ContainerRegistryConfigHarbor struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type ContainerRegistryConfig struct {
	Harbor *ContainerRegistryConfigHarbor `json:"harbor"`
}

type ContainerRegistryCreateInput struct {
	apis.StatusInfrasResourceBaseCreateInput

	// Repo type
	// required: true
	// enum: harbor
	Type ContainerRegistryType `json:"type"`

	// Repo URL
	// required: true
	// example: https://10.127.190.187
	Url string `json:"url"`

	// Configuration info
	Config ContainerRegistryConfig `json:"config"`
}

type ContainerRegistryUploadImageInput struct {
	// Repository is the path on server, e.g. 'yunion/influxdb'
	Repository string `json:"repository"`
	// Tag is image tag
	Tag string `json:"tag"`
}

type ContainerRegistryGetImageTagsInput struct {
	Repository string `json:"repository"`
}
