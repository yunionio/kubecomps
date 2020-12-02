package api

type VirtualMachine struct {
	ObjectTypeMeta

	// Hypervisor is virtual machine hypervisor
	Hypervisor string `json:"hypervisor"`
	// VcpuCount represents the number of CPUs of the virtual machine
	VcpuCount *int64 `json:"vcpuCount,omitempty"`
	// VmemSizeGB reprensents the size of memory
	VmemSizeGB *int64 `json:"vmemSizeGB,omitempty"`
	// InstanceType describes the specifications of the virtual machine
	InstanceType string `json:"instanceType,omitempty"`

	VirtualMachineStatus
}

type OnecloudExternalInfoBase struct {
	// Id is resource cloud resource id
	Id string `json:"id"`
	// Status is resource cloud status
	Status string `json:"status"`
	// Action indicate the latest action for external vm
	Action string `json:"action"`
	// Eip is elastic ip address
	Eip string `json:"eip,omitempty"`
}

type VirtualMachineInfo struct {
	OnecloudExternalInfoBase
	// Ips is internal attached ip addresses
	Ips []string `json:"ips"`
}

type VirtualMachineStatus struct {
	Status       string             `json:"status"`
	Reason       string             `json:"reason"`
	ExternalInfo VirtualMachineInfo `json:"externalInfo"`
	// TryTimes record the continuous creation try times
	TryTimes int32 `json:"tryTimes"`
}
