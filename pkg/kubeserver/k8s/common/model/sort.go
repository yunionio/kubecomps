package model

import (
	"reflect"
	"strings"

	"yunion.io/x/pkg/util/reflectutils"
)

type OrderType string

const (
	OrderDESC OrderType = "desc"
	OrderASC  OrderType = "asc"
)

type OrderField struct {
	Field IOrderField
	Order OrderType
}

func NewOrderField(f IOrderField, order OrderType) OrderField {
	return OrderField{
		Field: f,
		Order: order,
	}
}

type IOrderField interface {
	GetFieldName() string
	// default compare order is desc
	Compare(obj1, obj2 IK8sModel) bool
}

type OrderFields map[string]IOrderField

func (of OrderFields) Get(fieldName string) IOrderField {
	return of[fieldName]
}

func (of OrderFields) Set(fields ...IOrderField) OrderFields {
	for _, f := range fields {
		of[f.GetFieldName()] = f
	}
	return of
}

type OrderFieldCreationTimestamp struct{}

func (_ OrderFieldCreationTimestamp) GetFieldName() string {
	return "creationTimestamp"
}

func (_ OrderFieldCreationTimestamp) Compare(obj1, obj2 IK8sModel) bool {
	m1, _ := obj1.GetObjectMeta()
	m2, _ := obj2.GetObjectMeta()
	return !m1.CreationTimestamp.Before(&m2.CreationTimestamp)
}

func OrderFieldName() IOrderField {
	return OrderFieldFactoryForString("name")
}

func OrderFieldNamespace() IOrderField {
	return OrderFieldFactoryForString("namespace")
}

func OrderFieldStatus() IOrderField {
	return OrderFieldFactoryForString("status")
}

type orderFieldStringF struct {
	name string
}

func (f orderFieldStringF) GetFieldName() string {
	return f.name
}

func (f orderFieldStringF) Compare(obj1, obj2 IK8sModel) bool {
	meta1, _ := obj1.GetObjectMeta()
	meta2, _ := obj2.GetObjectMeta()
	v1 := reflect.ValueOf(meta1)
	v2 := reflect.ValueOf(meta2)
	name1, _ := reflectutils.FindStructFieldInterface(v1, f.name)
	name2, _ := reflectutils.FindStructFieldInterface(v2, f.name)

	return strings.Compare(name1.(string), name2.(string)) < 0
}

func OrderFieldFactoryForString(name string) IOrderField {
	return orderFieldStringF{name}
}
