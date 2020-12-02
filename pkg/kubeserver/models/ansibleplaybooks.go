package models

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"yunion.io/x/jsonutils"
	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/k8s/common/model"
)

var (
	ansiblePlaybookManager *SAnsiblePlaybookManager
)

func init() {
	GetAnsiblePlaybookManager()
}

type SAnsiblePlaybookManager struct {
	model.SK8sNamespaceResourceBaseManager
}

type SAnsiblePlaybook struct {
	model.SK8sNamespaceResourceBase
	UnstructuredResourceBase
}

func GetAnsiblePlaybookManager() *SAnsiblePlaybookManager {
	if ansiblePlaybookManager == nil {
		ansiblePlaybookManager = &SAnsiblePlaybookManager{
			SK8sNamespaceResourceBaseManager: model.NewK8sNamespaceResourceBaseManager(new(SAnsiblePlaybook), "k8s_ansibleplaybook", "k8s_ansibleplaybooks"),
		}
		ansiblePlaybookManager.SetVirtualObject(ansiblePlaybookManager)
		RegisterK8sModelManager(ansiblePlaybookManager)
	}
	return ansiblePlaybookManager
}

func (m *SAnsiblePlaybookManager) GetK8sResourceInfo() model.K8sResourceInfo {
	return model.K8sResourceInfo{
		ResourceName: api.ResourceNameAnsiblePlaybook,
		KindName:     api.KindNameAnsiblePlaybook,
		Object:       &unstructured.Unstructured{},
	}
}

func (obj *SAnsiblePlaybook) GetAPIObject() (*api.AnsiblePlaybook, error) {
	out := new(api.AnsiblePlaybook)
	if err := obj.ConvertToAPIObject(obj, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (obj *SAnsiblePlaybook) FillAPIObjectBySpec(specObj jsonutils.JSONObject, out IUnstructuredOutput) error {
	ret := out.(*api.AnsiblePlaybook)
	if tmplateName, err := specObj.GetString("playbookTemplateRef", "name"); err == nil {
		ret.PlaybookTemplateRef = &api.LocalObjectReference{
			Name: tmplateName,
		}
	}
	if maxRetryTime, _ := specObj.Int("maxRetryTimes"); maxRetryTime > 0 {
		mrt := int32(maxRetryTime)
		ret.MaxRetryTime = &mrt
	}
	return nil
}

func (obj *SAnsiblePlaybook) FillAPIObjectByStatus(statusObj jsonutils.JSONObject, out IUnstructuredOutput) error {
	ret := out.(*api.AnsiblePlaybook)
	phase, _ := statusObj.GetString("phase")
	ret.AnsiblePlaybookStatus.Status = phase

	tryTimes, _ := statusObj.Int("tryTimes")
	ret.TryTimes = int32(tryTimes)

	extInfo := &ret.AnsiblePlaybookStatus.ExternalInfo
	if extraInfoObj, err := statusObj.Get("externalInfo"); err == nil {
		extraInfoObj.Unmarshal(extInfo)
	}
	return nil
}
