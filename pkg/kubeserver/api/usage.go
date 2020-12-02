package api

type GlobalUsage struct {
	AllUsage     *UsageResult `json:"all"`
	DomainUsage  *UsageResult `json:"domain"`
	ProjectUsage *UsageResult `json:"project"`
}

type UsageResult struct {
	ClusterUsage *ClusterUsage `json:"cluster"`
}

type MemoryUsage struct {
	// memory total capacity
	Capacity int64 `json:"capacity"`
	// memory pods request size
	Request int64 `json:"request"`
	// memory pods limit size
	Limit int64 `json:"limit"`
}

func (u *MemoryUsage) Add(ou *MemoryUsage) *MemoryUsage {
	u.Capacity += ou.Capacity
	u.Request += ou.Request
	u.Limit += ou.Limit
	return u
}

type CpuUsage struct {
	// cpu total capacity
	Capacity int64 `json:"capacity"`
	// cpu pods request millcore
	Request int64 `json:"request"`
	// cpu pods limit millcore
	Limit int64 `json:"limit"`
}

func (u *CpuUsage) Add(ou *CpuUsage) *CpuUsage {
	u.Capacity += ou.Capacity
	u.Request += ou.Request
	u.Limit += ou.Limit
	return u
}

type PodUsage struct {
	// pod creatable count capacity
	Capacity int64 `json:"capacity"`
	// pod used total count
	Count int64 `json:"count"`
}

func (u *PodUsage) Add(ou *PodUsage) *PodUsage {
	u.Capacity += ou.Capacity
	u.Count += ou.Count
	return u
}

type NodeUsage struct {
	// node memory usage
	Memory *MemoryUsage `json:"memory"`
	// node cpu usage
	Cpu *CpuUsage `json:"cpu"`
	// node pod usage
	Pod *PodUsage `json:"pod"`
	// node count
	Count int64 `json:"count"`
	// node ready count
	ReadyCount int64 `json:"ready_count"`
	// node not ready count
	NotReadyCount int64 `json:"not_ready_count"`
}

func (u *NodeUsage) Add(ou *NodeUsage) *NodeUsage {
	u.Memory.Add(ou.Memory)
	u.Cpu.Add(ou.Cpu)
	u.Pod.Add(ou.Pod)
	u.Count += ou.Count
	u.ReadyCount += ou.ReadyCount
	u.NotReadyCount += ou.NotReadyCount
	return u
}

func NewNodeUsage() *NodeUsage {
	return &NodeUsage{
		Memory: new(MemoryUsage),
		Cpu:    new(CpuUsage),
		Pod:    new(PodUsage),
	}
}

type ClusterUsage struct {
	// node usage
	Node *NodeUsage `json:"node"`
	// cluster count
	Count int64 `json:"count"`
}
