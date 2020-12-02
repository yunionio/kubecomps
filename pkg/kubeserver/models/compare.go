package models

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/pkg/errors"
)

type valueElement struct {
	key   string
	value reflect.Value
}

type valueSet []valueElement

func (v valueSet) Len() int {
	return len(v)
}

func (v valueSet) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func (v valueSet) Less(i, j int) bool {
	return strings.Compare(v[i].key, v[j].key) < 0
}

func valueSet2Array(dbSet interface{}, fieldFunc func(obj interface{}) string) ([]valueElement, error) {
	dbSetValue := reflect.Indirect(reflect.ValueOf(dbSet))
	if dbSetValue.Kind() != reflect.Slice {
		return nil, fmt.Errorf("input set is not a slice")
	}
	ret := make([]valueElement, dbSetValue.Len())
	for i := 0; i < dbSetValue.Len(); i += 1 {
		val := dbSetValue.Index(i)
		keyVal := fieldFunc(val.Interface())
		ret[i] = valueElement{value: dbSetValue.Index(i), key: keyVal}
	}
	return ret, nil
}

func CompareSetsByFunc(
	localObjs []IClusterModel,
	getExternalIdMethod func(localObj interface{}) string,
	extObjs interface{},
	getGlobalIdMethod func(remoteObj interface{}) string,
	removed interface{},
	commonDB interface{},
	commonExt interface{},
	added interface{}) error {
	dbSetArray, err := valueSet2Array(localObjs, getExternalIdMethod)
	if err != nil {
		return errors.Wrapf(err, "Get local objects %s", getExternalIdMethod)
	}
	extSetArray, err := valueSet2Array(extObjs, getGlobalIdMethod)
	if err != nil {
		return errors.Wrapf(err, "Get remote objects %s", getGlobalIdMethod)
	}
	sort.Sort(valueSet(dbSetArray))
	sort.Sort(valueSet(extSetArray))

	removedValue := reflect.Indirect(reflect.ValueOf(removed))
	commonDBValue := reflect.Indirect(reflect.ValueOf(commonDB))
	commonExtValue := reflect.Indirect(reflect.ValueOf(commonExt))
	addedValue := reflect.Indirect(reflect.ValueOf(added))

	i := 0
	j := 0
	for i < len(dbSetArray) || j < len(extSetArray) {
		if i < len(dbSetArray) && j < len(extSetArray) {
			cmp := strings.Compare(dbSetArray[i].key, extSetArray[j].key)
			if cmp == 0 {
				newVal1 := reflect.Append(commonDBValue, dbSetArray[i].value)
				commonDBValue.Set(newVal1)
				newVal2 := reflect.Append(commonExtValue, extSetArray[j].value)
				commonExtValue.Set(newVal2)
				i += 1
				j += 1
			} else if cmp < 0 {
				newVal := reflect.Append(removedValue, dbSetArray[i].value)
				removedValue.Set(newVal)
				i += 1
			} else {
				newVal := reflect.Append(addedValue, extSetArray[j].value)
				addedValue.Set(newVal)
				j += 1
			}
		} else if i >= len(dbSetArray) {
			newVal := reflect.Append(addedValue, extSetArray[j].value)
			addedValue.Set(newVal)
			j += 1
		} else if j >= len(extSetArray) {
			newVal := reflect.Append(removedValue, dbSetArray[i].value)
			removedValue.Set(newVal)
			i += 1
		}
	}
	return nil
}

func CompareRemoteObjectSets(
	localObjs []IClusterModel,
	extObjs []interface{},
	getGlobalIdMethod func(remoteObj interface{}) string,
	removed interface{},
	localCommon interface{},
	extCommon interface{},
	added interface{}) error {
	return CompareSetsByFunc(
		localObjs, func(localObj interface{}) string {
			return localObj.(db.IExternalizedModel).GetExternalId()
		},
		extObjs, getGlobalIdMethod,
		removed, localCommon, extCommon, added)
}
