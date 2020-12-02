package helm

import (
	helmchart "k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/repo"
)

type ChartPackage struct {
	*helmchart.Chart
	Repo string `json:"repo"`
}

type SpotguideFile struct {
	Options []SpotguideOptions `json:"options"`
}

type SpotguideOptions struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Default bool   `json:"default"`
	Info    string `json:"info"`
	Key     string `json:"key"`
}

type ChartDetail struct {
	Name    string             `json:"name"`
	Repo    string             `json:"repo"`
	Chart   *repo.ChartVersion `json:"chart"`
	Values  string             `json:"values"`
	Readme  string             `json:"readme"`
	Options []SpotguideOptions `json:"options"`
}
