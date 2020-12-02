package api

// LimitRange provides resource limit range values
type LimitRangeItem struct {
	// ResourceName usage constraints on this kind by resource name
	ResourceName string `json:"resourceName,omitempty"`
	// ResourceType of resource that this limit applies to
	ResourceType string `json:"resourceType,omitempty"`
	// Min usage constraints on this kind by resource name
	Min string `json:"min,omitempty"`
	// Max usage constraints on this kind by resource name
	Max string `json:"max,omitempty"`
	// Default resource requirement limit value by resource name.
	Default string `json:"default,omitempty"`
	// DefaultRequest resource requirement request value by resource name.
	DefaultRequest string `json:"defaultRequest,omitempty"`
	// MaxLimitRequestRatio represents the max burst value for the named resource
	MaxLimitRequestRatio string `json:"maxLimitRequestRatio,omitempty"`
}

type LimitRange struct {
	ObjectMeta
	TypeMeta
	// v1.LimitRangeSpec
	Limits []*LimitRangeItem `json:"limits"`
}

type LimitRangeDetail struct {
	LimitRange
}

type LimitRangeDetailV2 struct {
	NamespaceResourceDetail
	// v1.LimitRangeSpec
	Limits []*LimitRangeItem `json:"limits"`
}
