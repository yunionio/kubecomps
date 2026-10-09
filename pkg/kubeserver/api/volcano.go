package api

const (
	VolcanoJobAPIVersion       = "batch.volcano.sh/v1alpha1"
	VolcanoQueueAPIVersion     = "scheduling.volcano.sh/v1beta1"
	VolcanoPodGroupAPIVersion  = "scheduling.volcano.sh/v1beta1"
	VolcanoHyperNodeAPIVersion = "topology.volcano.sh/v1alpha1"

	VolcanoJobKind       = "Job"
	VolcanoQueueKind     = "Queue"
	VolcanoPodGroupKind  = "PodGroup"
	VolcanoHyperNodeKind = "HyperNode"
)

type VCJobContainerInput struct {
	Name            string `json:"name"`
	Image           string `json:"image"`
	ImagePullPolicy string `json:"imagePullPolicy"`
	Command         string `json:"command"`
	Cpu             string `json:"cpu"`
	CpuLimit        string `json:"cpuLimit"`
	Memory          string `json:"memory"`
	MemoryLimit     string `json:"memoryLimit"`
}

type VCJobTaskInput struct {
	Name          string                `json:"name"`
	Replicas      int32                 `json:"replicas"`
	RestartPolicy string                `json:"restartPolicy"`
	Containers    []VCJobContainerInput `json:"containers"`
}

type VCJobCreateInput struct {
	NamespaceResourceCreateInput

	Queue               string           `json:"queue"`
	MinAvailable        int32            `json:"minAvailable"`
	SchedulerName       string           `json:"schedulerName"`
	PriorityClassName   string           `json:"priorityClassName"`
	RestartOnEvict      bool             `json:"restartOnEvict"`
	NetworkTopologyMode string           `json:"networkTopologyMode"`
	HighestTierAllowed  int32            `json:"highestTierAllowed"`
	Tasks               []VCJobTaskInput `json:"tasks"`
}

type VCJobListInput struct {
	NamespaceResourceListInput

	Queue string `json:"queue"`
}

type VCJobDetail struct {
	NamespaceResourceDetail

	Queue               string `json:"queue"`
	PodGroupName        string `json:"podGroupName"`
	Status              string `json:"status"`
	MinAvailable        int64  `json:"minAvailable"`
	PriorityClassName   string `json:"priorityClassName"`
	NetworkTopologyMode string `json:"networkTopologyMode"`
	HighestTierAllowed  int64  `json:"highestTierAllowed"`
	Running             int64  `json:"running"`
	Pending             int64  `json:"pending"`
}

type VCHyperNodeMemberInput struct {
	Type       string `json:"type"`
	LabelKey   string `json:"labelKey"`
	LabelValue string `json:"labelValue"`
	ExactMatch string `json:"exactMatch"`
}

type VCHyperNodeCreateInput struct {
	ClusterResourceCreateInput

	Tier     int32                    `json:"tier"`
	TierName string                   `json:"tierName"`
	Members  []VCHyperNodeMemberInput `json:"members"`
}

type VCHyperNodeDetail struct {
	ClusterResourceDetail

	Tier       int64    `json:"tier"`
	TierName   string   `json:"tierName"`
	NodeCount  int64    `json:"nodeCount"`
	Status     string   `json:"status"`
	Nodes      []string `json:"nodes,omitempty"`
	HyperNodes []string `json:"hyperNodes,omitempty"`
}

type VCQueueSpecInput struct {
	Weight      int32  `json:"weight"`
	Priority    int32  `json:"priority"`
	Reclaimable *bool  `json:"reclaimable"`
	Parent      string `json:"parent"`

	GuaranteeCpu     string `json:"guaranteeCpu"`
	GuaranteeMemory  string `json:"guaranteeMemory"`
	CapabilityCpu    string `json:"capabilityCpu"`
	CapabilityMemory string `json:"capabilityMemory"`
	DeservedCpu      string `json:"deservedCpu"`
	DeservedMemory   string `json:"deservedMemory"`
}

type VCQueueCreateInput struct {
	ClusterResourceCreateInput
	VCQueueSpecInput
}

type VCQueueUpdateInput struct {
	ClusterResourceUpdateInput
	VCQueueSpecInput
}

type VCQueueDetail struct {
	ClusterResourceDetail

	Parent           string `json:"parent"`
	Weight           int64  `json:"weight"`
	Priority         int64  `json:"priority"`
	Reclaimable      bool   `json:"reclaimable"`
	GuaranteeCpu     string `json:"guaranteeCpu"`
	GuaranteeMemory  string `json:"guaranteeMemory"`
	CapabilityCpu    string `json:"capabilityCpu"`
	CapabilityMemory string `json:"capabilityMemory"`
	DeservedCpu      string `json:"deservedCpu"`
	DeservedMemory   string `json:"deservedMemory"`
	Status           string `json:"status"`
}

type VCPodGroupListInput struct {
	NamespaceResourceListInput

	Queue string `json:"queue"`
}

type VCPodGroupDetail struct {
	NamespaceResourceDetail

	Queue     string `json:"queue"`
	Job       string `json:"job"`
	MinMember int64  `json:"minMember"`
	Status    string `json:"status"`
}
