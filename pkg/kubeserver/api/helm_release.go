package api

import (
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/time"

	"yunion.io/x/jsonutils"
)

const (
	ReleaseStatusDeploying  = "deploying"
	ReleaseStatusDeployFail = "deploy_fail"
	ReleaseStatusDeployed   = "deployed"
	ReleaseStatusUpdating   = "updating"
	ReleaseStatusUpdateFail = "update_fail"
	ReleaseStatusDeleting   = "deleting"
	ReleaseStatusDeleteFail = "delete_fail"
)

type ReleaseCreateInput struct {
	NamespaceResourceCreateInput
	Repo string `json:"repo"`
	// Deprecated, use Chart and Repo
	ChartName string `json:"chart_name"`
	Chart     string `json:"chart"`
	// Deprecated, use name
	ReleaseName string `json:"release_name"`
	Version     string `json:"version"`
	// Values is yaml config content
	Values     string               `json:"values"`
	Sets       map[string]string    `json:"sets"`
	ValuesJson jsonutils.JSONObject `json:"values_json"`
}

type ReleaseUpdateInput struct {
	ReleaseCreateInput
	RecreatePods bool `json:"recreate_pods"`
	// force resource updates through a replacement strategy
	Force bool `json:"force"`
	// when upgrading, reset the values to the ones built into the chart
	ResetValues bool `json:"reset_values"`
	// when upgrading, reuse the last release's values and merge in any overrides, if reset_values is specified, this is ignored
	ReUseValues bool `json:"reuse_values"`
}

type ReleaseListQuery struct {
	Filter       string `json:"filter"`
	All          bool   `json:"all"`
	AllNamespace bool   `json:"all_namespace"`
	Namespace    string `json:"namespace"`
	Admin        bool   `json:"admin"`
	Deployed     bool   `json:"deployed"`
	Deleted      bool   `json:"deleted"`
	Deleting     bool   `json:"deleting"`
	Failed       bool   `json:"failed"`
	Superseded   bool   `json:"superseded"`
	Pending      bool   `json:"pending"`
}

type Release struct {
	*release.Release
	*ClusterMeta
	Status     string            `json:"status"`
	PodsStatus map[string]string `json:"pods_status"`
}

type ReleaseDetail struct {
	Release
	Resources map[string][]interface{} `json:"resources"`
	Files     []*chart.File            `json:"files"`
}

type ReleaseHistoryInput struct {
	Max int `json:"max"`
}

type ReleaseHistoryInfo struct {
	Revision    int       `json:"revision"`
	Updated     time.Time `json:"updated"`
	Status      string    `json:"status"`
	Chart       string    `json:"chart"`
	Description string    `json:"description"`
}

type ReleaseRollbackInput struct {
	Revision    int    `json:"revision"`
	Description string `json:"description"`
	// will (if true) recreate pods after a rollback.
	Recreate bool `json:"recreate"`
	// will (if true) force resource upgrade through uninstall/recreate if needed
	Force bool `json:"force"`
}

type ReleaseListInputV2 struct {
	NamespaceResourceListInput
	// Release type
	// enum: internal, external
	Type string `json:"type"`
}

type ReleaseV2 struct {
	NamespaceResourceDetail
	// Info provides information about a release
	Info *release.Info `json:"info,omitempty"`
	// ChartInfo is the chart that was released.
	ChartInfo *chart.Chart `json:"chart_info,omitempty"`
	// Config is the set of extra Values added to the chart.
	// These values override the default values inside of the chart.
	Config map[string]interface{} `json:"config,omitempty"`
	// Manifest is the string representation of the rendered template.
	Manifest string `json:"manifest,omitempty"`
	// Hooks are all of the hooks declared for this release.
	Hooks []*release.Hook `json:"hooks,omitempty"`
	// Version is an int which represents the version of the release.
	Version int `json:"version,omitempty"`

	PodsStatus map[string]string `json:"pods_status,omitempty"`
}

type ReleaseDetailV2 struct {
	ReleaseV2
	Type      RepoType                 `json:"type"`
	Resources map[string][]interface{} `json:"resources"`
	Files     []*chart.File            `json:"files"`
}
