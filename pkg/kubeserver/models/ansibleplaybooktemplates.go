package models

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"yunion.io/x/jsonutils"
	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/k8s/common/model"
)

var (
	ansiblePlaybookTemplateManager *SAnsiblePlaybookTemplateManager
)

func init() {
	GetAnsiblePlaybookTemplateManager()
}

func GetAnsiblePlaybookTemplateManager() *SAnsiblePlaybookTemplateManager {
	if ansiblePlaybookTemplateManager == nil {
		ansiblePlaybookTemplateManager = &SAnsiblePlaybookTemplateManager{
			SK8sNamespaceResourceBaseManager: model.NewK8sNamespaceResourceBaseManager(new(SAnsiblePlaybook), "k8s_ansibleplaybooktemplate", "k8s_ansibleplaybooktemplates"),
		}
		ansiblePlaybookTemplateManager.SetVirtualObject(ansiblePlaybookTemplateManager)
		RegisterK8sModelManager(ansiblePlaybookTemplateManager)
	}
	return ansiblePlaybookTemplateManager
}

type SAnsiblePlaybookTemplateManager struct {
	model.SK8sNamespaceResourceBaseManager
}

type SAnsiblePlaybookTemplate struct {
	model.SK8sNamespaceResourceBase
	UnstructuredResourceBase
}

func (m *SAnsiblePlaybookTemplateManager) GetK8sResourceInfo() model.K8sResourceInfo {
	return model.K8sResourceInfo{
		ResourceName: api.ResourceNameAnsiblePlaybookTemplate,
		KindName:     api.KindNameAnsiblePlaybookTemplate,
		Object:       &unstructured.Unstructured{},
	}
}

func (obj *SAnsiblePlaybookTemplate) GetAPIObject() (*api.AnsiblePlaybookTemplate, error) {
	out := new(api.AnsiblePlaybookTemplate)
	if err := obj.ConvertToAPIObject(obj, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (obj *SAnsiblePlaybookTemplate) FillAPIObjectBySpec(specObj jsonutils.JSONObject, out IUnstructuredOutput) error {
	ret := out.(*api.AnsiblePlaybookTemplate)
	if err := specObj.Unmarshal(&ret.AnsiblePlaybookTemplateSpec); err != nil {
		return err
	}
	return nil
}

func (obj *SAnsiblePlaybookTemplate) FillAPIObjectByStatus(statusObj jsonutils.JSONObject, out IUnstructuredOutput) error {
	return nil
}
