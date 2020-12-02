// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model

import (
	"encoding/json"
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
)

const (
	DMethodValidateCreateData       = "ValidateCreateData"
	DMethodValidateUpdateData       = "ValidateUpdateData"
	DMethodValidateDeleteCondition  = "ValidateDeleteCondition"
	DMethodNewK8SRawObjectForCreate = "NewK8SRawObjectForCreate"
	DMethodNewK8SRawObjectForUpdate = "NewK8SRawObjectForUpdate"
	DMethodCustomizeDelete          = "CustomizeDelete"
	DMethodListItemFilter           = "ListItemFilter"
	DMethodGetAPIObject             = "GetAPIObject"
	DMethodGetAPIDetailObject       = "GetAPIDetailObject"
)

type Caller struct {
	modelVal reflect.Value
	funcName string
	inputs   []interface{}

	funcVal reflect.Value
}

func NewCaller(model interface{}, fName string) *Caller {
	return &Caller{
		modelVal: reflect.ValueOf(model),
		funcName: fName,
	}
}

func (c *Caller) Inputs(inputs ...interface{}) *Caller {
	c.inputs = inputs
	return c
}

func (c *Caller) Call() ([]reflect.Value, error) {
	return callObject(c.modelVal, c.funcName, c.inputs...)
}

func call(obj interface{}, fName string, inputs ...interface{}) ([]reflect.Value, error) {
	return callObject(reflect.ValueOf(obj), fName, inputs...)
}

func FindFunc(modelVal reflect.Value, fName string) (reflect.Value, error) {
	funcVal := modelVal.MethodByName(fName)
	if !funcVal.IsValid() || funcVal.IsNil() {
		log.Debugf("find method %s for %s", fName, modelVal.Type())
		if modelVal.Kind() != reflect.Ptr {
			return funcVal, errors.Wrapf(httperrors.ErrNotImplemented, "%s not implemented", fName)
		}
		modelVal = modelVal.Elem()
		if modelVal.Kind() != reflect.Struct {
			return funcVal, errors.Wrapf(httperrors.ErrNotImplemented, "%s not implemented", fName)
		}
		modelType := modelVal.Type()
		for i := 0; i < modelType.NumField(); i += 1 {
			fieldType := modelType.Field(i)
			if fieldType.Anonymous {
				fieldValue := modelVal.Field(i)
				if fieldValue.Kind() != reflect.Ptr && fieldValue.CanAddr() {
					newFuncVal, err := FindFunc(fieldValue.Addr(), fName)
					if err == nil {
						if !funcVal.IsValid() || funcVal.IsNil() {
							funcVal = newFuncVal
						} else {
							return funcVal, errors.Wrapf(httperrors.ErrNotSupported, "%s is ambiguous", fName)
						}
					}
				} else if fieldValue.Kind() == reflect.Ptr {
					newFuncVal, err := FindFunc(fieldValue, fName)
					if err == nil {
						if !funcVal.IsValid() || funcVal.IsNil() {
							funcVal = newFuncVal
						} else {
							return funcVal, errors.Wrapf(httperrors.ErrNotSupported, "%s is ambiguous", fName)
						}
					}
				}
			}
		}
		if !funcVal.IsValid() || funcVal.IsNil() {
			return funcVal, errors.Wrapf(httperrors.ErrNotImplemented, "%s is not implemented", fName)
		}
	}
	return funcVal, nil
}

func callObject(modelVal reflect.Value, fName string, inputs ...interface{}) ([]reflect.Value, error) {
	funcVal := modelVal.MethodByName(fName)
	return callFunc(funcVal, fName, inputs...)
}

func callFunc(funcVal reflect.Value, fName string, inputs ...interface{}) ([]reflect.Value, error) {
	if !funcVal.IsValid() || funcVal.IsNil() {
		return nil, httperrors.NewActionNotFoundError(fmt.Sprintf("%s method not found", fName))
	}
	funcType := funcVal.Type()
	paramLen := funcType.NumIn()
	if paramLen != len(inputs) {
		return nil, httperrors.NewInternalServerError("%s method params length not match, expected %d, input %d", fName, paramLen, len(inputs))
	}
	params := make([]*param, paramLen)
	for i := range inputs {
		params[i] = newParam(funcType.In(i), inputs[i])
	}
	args := convertParams(params)
	return funcVal.Call(args), nil
}

func convertParams(params []*param) []reflect.Value {
	ret := make([]reflect.Value, 0)
	for _, p := range params {
		ret = append(ret, p.convert())
	}
	return ret
}

type param struct {
	pType reflect.Type
	input interface{}
}

func newParam(pType reflect.Type, input interface{}) *param {
	return &param{
		pType: pType,
		input: input,
	}
}

func isJSONObject(input interface{}) (jsonutils.JSONObject, bool) {
	val := reflect.ValueOf(input)
	obj, ok := val.Interface().(jsonutils.JSONObject)
	if !ok {
		return nil, false
	}
	return obj, true
}

func (p *param) convert() reflect.Value {
	if p.input == nil {
		return reflect.New(p.pType).Elem()
	}
	obj, ok := isJSONObject(p.input)
	if !ok {
		return reflect.ValueOf(p.input)
	}
	// generate object by type
	val := reflect.New(p.pType)
	obj.Unmarshal(val.Interface())
	return val.Elem()
}

func K8sObjectToJSONObject(obj runtime.Object) jsonutils.JSONObject {
	ov := reflect.ValueOf(obj)
	return ValueToJSONDict(ov)
}

func ValueToJSONObject(out reflect.Value) jsonutils.JSONObject {
	if gotypes.IsNil(out.Interface()) {
		return nil
	}

	if obj, ok := isJSONObject(out); ok {
		return obj
	}
	jsonBytes, err := json.Marshal(out.Interface())
	if err != nil {
		panic(fmt.Sprintf("marshal json: %v", err))
	}
	jObj, err := jsonutils.Parse(jsonBytes)
	if err != nil {
		panic(fmt.Sprintf("jsonutils.Parse bytes: %s, error %v", jsonBytes, err))
	}
	return jObj
}

func ValueToJSONDict(out reflect.Value) *jsonutils.JSONDict {
	jsonObj := ValueToJSONObject(out)
	if jsonObj == nil {
		return nil
	}
	return jsonObj.(*jsonutils.JSONDict)
}

func ValueToError(out reflect.Value) error {
	errVal := out.Interface()
	if !gotypes.IsNil(errVal) {
		return errVal.(error)
	}
	return nil
}

func mergeInputOutputData(data *jsonutils.JSONDict, resVal reflect.Value) *jsonutils.JSONDict {
	retJson := ValueToJSONDict(resVal)
	// preserve the input info not returned by caller
	data.Update(retJson)
	return data
}

func ValidateCreateData(manager IK8sModelManager, ctx *RequestContext, query *jsonutils.JSONDict, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	ret, err := call(manager, DMethodValidateCreateData, ctx, query, data)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if len(ret) != 2 {
		return nil, httperrors.NewInternalServerError("Invalid ValidateCreateData return value")
	}
	resVal := ret[0]
	if err := ValueToError(ret[1]); err != nil {
		return nil, err
	}
	return mergeInputOutputData(data, resVal), nil
}

func NewK8SRawObjectForCreate(manager IK8sModelManager, ctx *RequestContext, data *jsonutils.JSONDict) (runtime.Object, error) {
	ret, err := call(manager, DMethodNewK8SRawObjectForCreate, ctx, data)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if len(ret) != 2 {
		return nil, httperrors.NewInternalServerError("Invalid NewK8SRawObjectForCreate return value")
	}
	if err := ValueToError(ret[1]); err != nil {
		return nil, err
	}
	return ret[0].Interface().(runtime.Object), nil
}

func ListItemFilter(ctx *RequestContext, manager IK8sModelManager, q IQuery, query *jsonutils.JSONDict) (IQuery, error) {
	ret, err := call(manager, DMethodListItemFilter, ctx, q, query)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if len(ret) != 2 {
		return nil, httperrors.NewInternalServerError("Invald ListItemFilter return value count %d", len(ret))
	}
	if err := ValueToError(ret[1]); err != nil {
		return nil, err
	}
	return ret[0].Interface().(IQuery), nil
}

func GetObject(model IK8sModel) (*jsonutils.JSONDict, error) {
	ret, err := call(model, DMethodGetAPIObject)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if len(ret) != 2 {
		return nil, httperrors.NewInternalServerError("Invalid GetExtraDetails return value count %d", len(ret))
	}
	if err := ValueToError(ret[1]); err != nil {
		return nil, err
	}
	return ValueToJSONDict(ret[0]), nil
}

func GetDetails(model IK8sModel) (*jsonutils.JSONDict, error) {
	ret, err := call(model, DMethodGetAPIDetailObject)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if len(ret) != 2 {
		return nil, httperrors.NewInternalServerError("Invalid GetExtraDetails return value count %d", len(ret))
	}
	if err := ValueToError(ret[1]); err != nil {
		return nil, err
	}
	return ValueToJSONDict(ret[0]), nil
}

/*func FetchCustomizeColumns(
	manager IModelManager,
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) ([]*jsonutils.JSONDict, error) {
	ret, err := call(manager, "FetchCustomizeColumns", ctx, userCred, query, objs, fields, isList)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if len(ret) != 1 {
		return nil, httperrors.NewInternalServerError("Invalid FetchCustomizeColumns return value count %d", len(ret))
	}
	if ret[0].IsNil() {
		return nil, nil
	}
	if ret[0].Kind() != reflect.Slice {
		return nil, httperrors.NewInternalServerError("Invalid FetchCustomizeColumns return value type, not a slice!")
	}
	if ret[0].Len() != len(objs) {
		return nil, httperrors.NewInternalServerError("Invalid FetchCustomizeColumns return value, inconsistent obj count: input %d != output %d", len(objs), ret[0].Len())
	}
	retVal := make([]*jsonutils.JSONDict, ret[0].Len())
	for i := 0; i < ret[0].Len(); i += 1 {
		jsonDict := ValueToJSONDict(ret[0].Index(i))
		jsonDict.Update(jsonutils.Marshal(objs[i]))
		retVal[i] = jsonDict
	}
	return retVal, nil
}*/

func ValidateUpdateData(model IK8sModel, ctx *RequestContext, query *jsonutils.JSONDict, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	ret, err := call(model, DMethodValidateUpdateData, ctx, query, data)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if len(ret) != 2 {
		return nil, httperrors.NewInternalServerError("Invalid ValidateUpdateData return value")
	}
	resVal := ret[0]
	if err := ValueToError(ret[1]); err != nil {
		return nil, err
	}
	return mergeInputOutputData(data, resVal), nil
}

func NewK8SRawObjectForUpdate(model IK8sModel, ctx *RequestContext, data *jsonutils.JSONDict) (runtime.Object, error) {
	ret, err := call(model, DMethodNewK8SRawObjectForUpdate, ctx, data)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if len(ret) != 2 {
		return nil, httperrors.NewInternalServerError("Invalid NewK8SRawObjectForCreate return value")
	}
	if err := ValueToError(ret[1]); err != nil {
		return nil, err
	}
	return ret[0].Interface().(runtime.Object), nil
}

func ValidateDeleteCondition(model IK8sModel, ctx *RequestContext, query, data *jsonutils.JSONDict) error {
	ret, err := call(model, DMethodValidateDeleteCondition, ctx, query, data)
	if err != nil {
		return httperrors.NewGeneralError(err)
	}
	if len(ret) != 1 {
		return httperrors.NewInternalServerError("Invald CustomizeDelete return value")
	}
	return ValueToError(ret[0])
}

func CustomizeDelete(model IK8sModel, ctx *RequestContext, query, data *jsonutils.JSONDict) error {
	ret, err := call(model, DMethodCustomizeDelete, ctx, query, data)
	if err != nil {
		return httperrors.NewGeneralError(err)
	}
	if len(ret) != 1 {
		return httperrors.NewInternalServerError("Invald CustomizeDelete return value")
	}
	return ValueToError(ret[0])
}
