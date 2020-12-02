package gotypes

import (
	"reflect"

	"yunion.io/x/pkg/gotypes"
)

func RegisterSerializable(objs ...gotypes.ISerializable) {
	for _, obj := range objs {
		tmp := obj
		gotypes.RegisterSerializable(reflect.TypeOf(tmp), func() gotypes.ISerializable {
			vpt := reflect.TypeOf(tmp)
			vt := vpt.Elem()
			return reflect.New(vt).Interface().(gotypes.ISerializable)
		})
	}
}
