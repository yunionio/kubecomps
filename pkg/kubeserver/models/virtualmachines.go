package models

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/version"

	"yunion.io/x/jsonutils"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/k8s/common/model"
)

var (
	virtualMachineManager *SVirtualMachineManager
)

func init() {
	GetVirtualMachineManager()
}

type SVirtualMachineManager struct {
	model.SK8sNamespaceResourceBaseManager
}

type SVirtualMachine struct {
	model.SK8sNamespaceResourceBase
	UnstructuredResourceBase
}

func GetVirtualMachineManager() *SVirtualMachineManager {
	if virtualMachineManager == nil {
		virtualMachineManager = &SVirtualMachineManager{
			SK8sNamespaceResourceBaseManager: model.NewK8sNamespaceResourceBaseManager(new(SVirtualMachine), "virtualmachine", "virtualmachines"),
		}
		virtualMachineManager.SetVirtualObject(virtualMachineManager)
		RegisterK8sModelManager(virtualMachineManager)
	}
	return virtualMachineManager
}

func (m *SVirtualMachineManager) GetK8sResourceInfo(version *version.Info) model.K8sResourceInfo {
	return model.K8sResourceInfo{
		ResourceName: api.ResourceNameVirtualMachine,
		KindName:     api.KindNameVirtualMachine,
		Object:       &unstructured.Unstructured{},
	}
}

func (obj *SVirtualMachine) GetAPIObject() (*api.VirtualMachine, error) {
	vm := new(api.VirtualMachine)
	if err := obj.ConvertToAPIObject(obj, vm); err != nil {
		return nil, err
	}
	return vm, nil
}

func (obj *SVirtualMachine) FillAPIObjectBySpec(specObj jsonutils.JSONObject, output IUnstructuredOutput) error {
	vm := output.(*api.VirtualMachine)
	vm.Hypervisor, _ = specObj.GetString("vmConfig", "hypervisor")
	if cpuCount, _ := specObj.Int("vmConfig", "vcpuCount"); cpuCount != 0 {
		vm.VcpuCount = &cpuCount
	}
	if mem, _ := specObj.Int("vmConfig", "vmemSizeGB"); mem != 0 {
		vm.VmemSizeGB = &mem
	}
	instanceType, _ := specObj.GetString("vmConfig", "instanceType")
	vm.InstanceType = instanceType
	return nil
}

func (obj *SVirtualMachine) FillAPIObjectByStatus(statusObj jsonutils.JSONObject, output IUnstructuredOutput) error {
	vm := output.(*api.VirtualMachine)

	phase, _ := statusObj.GetString("phase")
	vm.VirtualMachineStatus.Status = phase

	reason, _ := statusObj.GetString("reason")
	vm.VirtualMachineStatus.Reason = reason

	tryTimes, _ := statusObj.Int("tryTimes")
	vm.VirtualMachineStatus.TryTimes = int32(tryTimes)

	extInfo := &vm.VirtualMachineStatus.ExternalInfo
	if extraInfoObj, err := statusObj.Get("externalInfo"); err == nil {
		extraInfoObj.Unmarshal(extInfo)
	}
	return nil
}
