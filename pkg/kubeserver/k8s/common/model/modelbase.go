package model

import (
	"encoding/json"
	"fmt"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/onecloud/pkg/cloudcommon/object"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
)

type SK8sObjectFactory struct {
	structType reflect.Type
}

func (f *SK8sObjectFactory) DataType() reflect.Type {
	return f.structType
}

func NewK8sObjectFactory(model interface{}) *SK8sObjectFactory {
	val := reflect.Indirect(reflect.ValueOf(model))
	st := val.Type()
	if st.Kind() != reflect.Struct {
		panic("expect struct kind")
	}
	factory := &SK8sObjectFactory{
		structType: st,
	}
	return factory
}

type SK8sModelBase struct {
	object.SObject

	K8sObject runtime.Object `json:"rawObject"`

	manager IK8sModelManager
	cluster ICluster
}

func (m SK8sModelBase) GetId() string {
	return ""
}

func (m SK8sModelBase) GetName() string {
	return ""
}

func (m SK8sModelBase) Keyword() string {
	return m.GetModelManager().Keyword()
}

func (m SK8sModelBase) KeywordPlural() string {
	return m.GetModelManager().KeywordPlural()
}

func (m *SK8sModelBase) SetModelManager(man IK8sModelManager, virtual IK8sModel) IK8sModel {
	m.manager = man
	m.SetVirtualObject(virtual)
	return m
}

func (m SK8sModelBase) GetModelManager() IK8sModelManager {
	return m.manager
}

func (m *SK8sModelBase) SetK8sObject(obj runtime.Object) IK8sModel {
	m.K8sObject = obj
	return m
}

func (m *SK8sModelBase) GetK8sObject() runtime.Object {
	return m.K8sObject
}

func (m *SK8sModelBase) GetMetaObject() metav1.Object {
	return m.GetK8sObject().(metav1.Object)
}

func (m *SK8sModelBase) SetCluster(cluster ICluster) IK8sModel {
	m.cluster = cluster
	return m
}

func (m *SK8sModelBase) GetCluster() ICluster {
	return m.cluster
}

func (m *SK8sModelBase) GetNamespace() string {
	return ""
}

func NewObjectMeta(kObj runtime.Object, cluster api.ICluster) (api.ObjectMeta, error) {
	unstructObj, isUnstruct := kObj.(runtime.Unstructured)
	meta := metav1.ObjectMeta{}
	if isUnstruct {
		metaObj := unstructObj.UnstructuredContent()["metadata"]
		if metaObj == nil {
			return api.ObjectMeta{}, errors.Error("unstructed object not contains metadata")
		}
		metaBytes, err := json.Marshal(metaObj)
		if err != nil {
			return api.ObjectMeta{}, errors.Wrap(err, "json.Marshal object")
		}
		if err := json.Unmarshal(metaBytes, &meta); err != nil {
			return api.ObjectMeta{}, errors.Wrap(err, "json unmarshal")
		}
	} else {
		v := reflect.ValueOf(kObj)
		f := reflect.Indirect(v).FieldByName("ObjectMeta")
		if !f.IsValid() {
			return api.ObjectMeta{}, errors.Errorf("get invalid object meta %#v", kObj)
		}
		meta = f.Interface().(metav1.ObjectMeta)
	}
	return api.ObjectMeta{
		ObjectMeta:  meta,
		ClusterMeta: api.NewClusterMeta(cluster),
	}, nil
}

func (m *SK8sModelBase) GetObjectMeta() (api.ObjectMeta, error) {
	kObj := m.GetK8sObject()
	return NewObjectMeta(kObj, m.GetCluster())
}

func (m *SK8sModelBase) GetTypeMeta() api.TypeMeta {
	kObj := m.GetK8sObject()
	unstructObj, isUnstruct := kObj.(runtime.Unstructured)
	meta := metav1.TypeMeta{}
	if isUnstruct {
		objContent := unstructObj.UnstructuredContent()
		if objContent == nil {
			panic(fmt.Sprintf("unstructed object not contains metadata"))
		}
		apiVersion := objContent["apiVersion"]
		kind := objContent["kind"]
		meta.APIVersion = apiVersion.(string)
		meta.Kind = kind.(string)
	} else {
		v := reflect.ValueOf(kObj)
		f := reflect.Indirect(v).FieldByName("TypeMeta")
		if !f.IsValid() {
			panic(fmt.Sprintf("get invalid object meta %#v", kObj))
		}
		meta = f.Interface().(metav1.TypeMeta)
	}
	return api.TypeMeta{
		TypeMeta: meta,
	}
}

type SK8sModelBaseManager struct {
	object.SObject

	factory     *SK8sObjectFactory
	orderFields OrderFields

	keyword       string
	keywordPlural string
}

func NewK8sModelBaseManager(model interface{}, keyword, keywordPlural string) SK8sModelBaseManager {
	factory := NewK8sObjectFactory(model)
	modelMan := SK8sModelBaseManager{
		factory:       factory,
		orderFields:   make(map[string]IOrderField),
		keyword:       keyword,
		keywordPlural: keywordPlural,
	}
	return modelMan
}

func (m *SK8sModelBaseManager) GetIModelManager() IK8sModelManager {
	virt := m.GetVirtualObject()
	if virt == nil {
		panic(fmt.Sprintf("Forgot to call SetVirtualObject?"))
	}
	r, ok := virt.(IK8sModelManager)
	if !ok {
		panic(fmt.Sprintf("Cannot convert virtual object to IK8SModelManager"))
	}
	return r
}

func (m *SK8sModelBaseManager) Factory() *SK8sObjectFactory {
	return m.factory
}

func (m *SK8sModelBaseManager) Keyword() string {
	return m.keyword
}

func (m *SK8sModelBaseManager) KeywordPlural() string {
	return m.keywordPlural
}

func (m *SK8sModelBaseManager) GetContextManagers() [][]IK8sModelManager {
	return nil
}

func (m *SK8sModelBaseManager) ValidateName(name string) error {
	return nil
}

func (m *SK8sModelBaseManager) GetQuery(cluster ICluster) IQuery {
	return NewK8SResourceQuery(cluster, m.GetIModelManager())
}

func (m *SK8sModelBaseManager) GetOrderFields() OrderFields {
	return m.orderFields
}

func (m *SK8sModelBaseManager) RegisterOrderFields(fields ...IOrderField) {
	m.orderFields.Set(fields...)
}

func (m *SK8sModelBaseManager) ListItemFilter(ctx *RequestContext, q IQuery, query api.ListInputK8SBase) (IQuery, error) {
	return q, nil
}
