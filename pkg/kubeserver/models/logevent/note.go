package logevent

import (
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
)

type BaseNote struct {
	DomainId  string    `json:"domain_id"`
	Domain    string    `json:"domain"`
	ClusterId string    `json:"cluster_id"`
	Cluster   string    `json:"cluster"`
	CreatedAt time.Time `json:"created_at"`

	Distribution string `json:"distribution"`
}

func NewBaseNote(domainId string, input api.ClusterResourceDetail) *BaseNote {
	return &BaseNote{
		DomainId:  domainId,
		Domain:    input.ProjectDomain,
		ClusterId: input.ClusterId,
		Cluster:   input.Cluster,
		CreatedAt: input.CreationTimestamp,

		Distribution: input.Distribution,
	}
}

type NamespaceResourceNote struct {
	*BaseNote
	Namespace       string            `json:"namespace"`
	NamespaceId     string            `json:"namespace_id"`
	NamespaceLabels map[string]string `json:"namespace_labels"`
}

func NewNamespaceResourceNote(domainId string, input api.NamespaceResourceDetail, nsLabels map[string]string) *NamespaceResourceNote {
	return &NamespaceResourceNote{
		BaseNote:        NewBaseNote(domainId, input.ClusterResourceDetail),
		Namespace:       input.Namespace,
		NamespaceId:     input.NamespaceId,
		NamespaceLabels: nsLabels,
	}
}

type Resources struct {
	CPU    int64 `json:"cpu"`
	Memory int64 `json:"memory"`
}

type PodNote struct {
	*NamespaceResourceNote
	Limits         *Resources `json:"limits"`
	Requests       *Resources `json:"requests"`
	CpuLimits      int        `json:"cpu_limits"`
	CpuRequests    int        `json:"cpu_requests"`
	MemoryLimits   int        `json:"memory_limits"`
	MemoryRequests int        `json:"memory_requests"`
	QOSClass       string     `json:"qosClass"`
	PodIP          string     `json:"pod_ip"`
	Status         string     `json:"status"`
}

func NewPodNote(domainId string,
	input api.PodDetailV2,
	nsLabels map[string]string,
	limits *Resources, requests *Resources) *PodNote {
	return &PodNote{
		NamespaceResourceNote: NewNamespaceResourceNote(domainId, input.NamespaceResourceDetail, nsLabels),
		Limits:                limits,
		CpuLimits:             int(limits.CPU),
		MemoryLimits:          int(limits.Memory),
		Requests:              requests,
		CpuRequests:           int(requests.CPU),
		MemoryRequests:        int(requests.Memory),
		QOSClass:              input.QOSClass,
		PodIP:                 input.PodIP,
		Status:                input.Status,
	}
}

type PVCNote struct {
	*NamespaceResourceNote
	CapacityMB   int    `json:"capacity_mb"`
	StorageClass string `json:"storage_class"`
}

func NewPVCNote(domainId string, input api.PersistentVolumeClaimDetail, nsLabels map[string]string) *PVCNote {
	quantity := input.Capacity[v1.ResourceStorage]
	return &PVCNote{
		NamespaceResourceNote: NewNamespaceResourceNote(domainId, input.NamespaceResourceDetail, nsLabels),
		CapacityMB:            int(quantity.ScaledValue(resource.Mega)),
		StorageClass:          *input.StorageClass,
	}
}
