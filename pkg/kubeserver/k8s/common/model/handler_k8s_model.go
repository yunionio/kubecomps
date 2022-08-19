package model

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/scheme"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
	"yunion.io/x/kubecomps/pkg/kubeserver/clientv2"
	"yunion.io/x/kubecomps/pkg/kubeserver/models/manager"
	"yunion.io/x/kubecomps/pkg/utils/k8serrors"
)

type RequestContext struct {
	ctx      context.Context
	userCred mcclient.TokenCredential
	cluster  ICluster
	query    *jsonutils.JSONDict
	data     *jsonutils.JSONDict
}

func NewRequestContext(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	cluster ICluster,
	query *jsonutils.JSONDict,
	data *jsonutils.JSONDict,
) *RequestContext {
	return &RequestContext{
		ctx:      ctx,
		userCred: userCred,
		cluster:  cluster,
		query:    query,
		data:     data,
	}
}

func (c *RequestContext) Context() context.Context {
	return c.ctx
}

func (c *RequestContext) Cluster() ICluster {
	return c.cluster
}

func (c *RequestContext) UserCred() mcclient.TokenCredential {
	return c.userCred
}

func (c *RequestContext) ShowAllNamespace() bool {
	return jsonutils.QueryBoolean(c.query, "all_namespace", false)
}

func (c *RequestContext) GetNamespaceByQuery() string {
	namespace, _ := c.query.GetString("namespace")
	return namespace
}

func (c *RequestContext) GetNamespaceByData() string {
	namespace, _ := c.data.GetString("namespace")
	return namespace
}

func (c *RequestContext) GetNamespace() string {
	ns := c.GetNamespaceByQuery()
	if ns == "" {
		ns = c.GetNamespaceByData()
	}
	return ns
}

func (c *RequestContext) GetQuery() *jsonutils.JSONDict {
	return c.query
}

func (c *RequestContext) GetData() *jsonutils.JSONDict {
	return c.data
}

type ICluster interface {
	api.ICluster

	GetHandler() client.ResourceHandler
	GetClientset() kubernetes.Interface
	GetClient() *clientv2.Client
	GetClusterObject() manager.ICluster
}

type K8SModelHandler struct {
	modelManager IK8sModelManager
}

func NewK8SModelHandler(manager IK8sModelManager) *K8SModelHandler {
	return &K8SModelHandler{modelManager: manager}
}

func (h *K8SModelHandler) Keyword() string {
	return h.modelManager.Keyword()
}

func (h *K8SModelHandler) KeywordPlural() string {
	return h.modelManager.KeywordPlural()
}

func (h *K8SModelHandler) Filter(f appsrv.FilterHandler) appsrv.FilterHandler {
	if consts.IsRbacEnabled() {
		return auth.AuthenticateWithDelayDecision(f, true)
	}
	return auth.Authenticate(f)
}

func (h *K8SModelHandler) List(ctx *RequestContext, query *jsonutils.JSONDict) (*modulebase.ListResult, error) {
	return ListK8SModels(ctx, h.modelManager, query)
}

func ListK8SModels(ctx *RequestContext, man IK8sModelManager, query *jsonutils.JSONDict) (*modulebase.ListResult, error) {
	var err error
	//var maxLimit int64 = consts.GetMaxPagingLimit()
	baseInput := new(api.ListInputK8SBase)
	if err := query.Unmarshal(baseInput); err != nil {
		return nil, err
	}
	limit := baseInput.Limit
	if limit == 0 {
		limit = consts.GetDefaultPagingLimit()
	}
	offset := baseInput.Offset
	// paginMarker := baseInput.PagingMarker

	q := man.GetQuery(ctx.Cluster()).Offset(offset).Limit(limit)
	q, err = ListItemFilter(ctx, man, q, query)
	if err != nil {
		return nil, err
	}

	listInput := new(api.ListInputK8SBase)
	query.Unmarshal(listInput)

	// process order by
	order := OrderDESC
	if listInput.Order == string(OrderASC) {
		order = OrderASC
	}
	orderByFields := make([]OrderField, 0)
	existsOrderFields := man.GetOrderFields()
	for _, fieldName := range listInput.OrderBy {
		if ret := existsOrderFields.Get(fieldName); ret != nil {
			orderByFields = append(orderByFields, OrderField{
				Field: ret,
				Order: order,
			})
		}
	}
	if len(orderByFields) == 0 {
		// add default order by creationTimestamp and name
		orderByFields = append(orderByFields,
			NewOrderField(OrderFieldCreationTimestamp{}, order),
			//NewOrderField(OrderFieldName(), order),
		)
	}
	q.AddOrderFields(orderByFields...)

	// process general filters
	if len(listInput.Filter) > 0 {
		for _, filter := range listInput.Filter {
			fc := ParseFilterClause(filter)
			if fc != nil {
				q.AddFilter(fc.QueryFilter())
			}
		}
	}

	listResult, err := Query2List(ctx, man, q)
	if err != nil {
		return nil, err
	}
	return calculateListResult(listResult, q.GetTotal(), q.GetLimit(), q.GetOffset()), nil
}

func calculateListResult(data []jsonutils.JSONObject, total, limit, offset int64) *modulebase.ListResult {
	ret := modulebase.ListResult{Data: data, Total: int(total), Limit: int(limit), Offset: int(offset)}
	return &ret
}

func Query2List(ctx *RequestContext, man IK8sModelManager, q IQuery) ([]jsonutils.JSONObject, error) {
	objs, err := q.FetchObjects()
	if err != nil {
		return nil, err
	}
	results := make([]jsonutils.JSONObject, len(objs))
	for i := range objs {
		jsonDict, err := GetObject(objs[i])
		if err != nil {
			return nil, err
		}
		results[i] = jsonDict
	}
	return results, nil
}

func (h *K8SModelHandler) Get(ctx *RequestContext, id string, query *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	namespace := ctx.GetNamespaceByQuery()
	model, err := fetchK8SModel(ctx, h.modelManager, namespace, id, query)
	if err != nil {
		return nil, err
	}

	/*if consts.IsRbacEnabled() {
		if err := db.IsObjectRbacAllowed(model, userCred, policy.PolicyActionGet); err != nil {
			return nil, err
		}
	} else if !model.AllowGetDetails(ctx, userCred, query) {
		return nil, httperrors.NewForbiddenError("Not allow to get details")
	}*/
	return getModelItemDetails(ctx, h.modelManager, model)
}

func getModelItemDetails(
	ctx *RequestContext,
	manager IK8sModelManager, item IK8sModel) (jsonutils.JSONObject, error) {
	return GetDetails(item)
}

func fetchK8SModel(
	ctx *RequestContext,
	man IK8sModelManager,
	namespace string,
	id string,
	query *jsonutils.JSONDict,
) (IK8sModel, error) {
	cluster := ctx.Cluster()
	version, err := cluster.GetClientset().Discovery().ServerVersion()
	if err != nil {
		return nil, err
	}
	cli := cluster.GetHandler()
	resInfo := man.GetK8sResourceInfo(version)
	obj, err := cli.Get(resInfo.ResourceName, namespace, id)
	if err != nil {
		return nil, err
	}
	model, err := NewK8SModelObject(man, cluster, obj)
	if err != nil {
		return nil, err
	}
	return model, nil
}

func NewK8SModelObjectByName(man IK8sModelManager, cluster ICluster, namespace, name string) (IK8sModel, error) {
	version, err := cluster.GetClientset().Discovery().ServerVersion()
	if err != nil {
		return nil, err
	}
	kind := man.GetK8sResourceInfo(version).ResourceName
	obj, err := cluster.GetHandler().Get(kind, namespace, name)
	if err != nil {
		return nil, err
	}
	return NewK8SModelObject(man, cluster, obj)
}

func NewK8SModelObject(man IK8sModelManager, cluster ICluster, obj runtime.Object) (IK8sModel, error) {
	m, ok := reflect.New(man.Factory().DataType()).Interface().(IK8sModel)
	if !ok {
		return nil, db.ErrInconsistentDataType
	}
	version, err := cluster.GetClientset().Discovery().ServerVersion()
	if err != nil {
		return nil, err
	}
	newObj := man.GetK8sResourceInfo(version).Object.DeepCopyObject()
	switch obj.(type) {
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.(*unstructured.Unstructured).Object, newObj); err != nil {
			return nil, err
		}
		obj = newObj
	}
	m.SetModelManager(man, m).SetCluster(cluster).SetK8sObject(obj)
	return m, nil
}

func NewK8SModelObjectByRef(
	man IK8sModelManager, cluster ICluster,
	ref *v1.ObjectReference) (IK8sModel, error) {
	obj, err := cluster.GetHandler().Get(ref.Kind, ref.Namespace, ref.Name)
	if err != nil {
		return nil, err
	}
	return NewK8SModelObject(man, cluster, obj)
}

func (h *K8SModelHandler) GetSpecific(ctx *RequestContext, id, spec string, query *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	namespace := ctx.GetNamespaceByQuery()
	model, err := fetchK8SModel(ctx, h.modelManager, namespace, id, query)
	if err != nil {
		return nil, err
	}

	specCamel := utils.Kebab2Camel(spec, "-")
	modelValue := reflect.ValueOf(model)

	/*if consts.IsRbacEnabled() {
		if err := db.IsObjectRbacAllowed(model, userCred, policy.PolicyActionGet); err != nil {
			return nil, err
		}
	} else if !model.AllowGetDetails(ctx, userCred, query) {
		return nil, httperrors.NewForbiddenError("Not allow to get details")
	}*/
	funcName := fmt.Sprintf("GetDetails%s", specCamel)
	outs, err := callObject(modelValue, funcName, ctx, query)
	if err != nil {
		return nil, err
	}
	resVal := outs[0]
	errVal := outs[1].Interface()
	if !gotypes.IsNil(errVal) {
		return nil, errVal.(error)
	}
	if gotypes.IsNil(resVal.Interface()) {
		return nil, nil
	}
	return ValueToJSONObject(resVal), nil
}

func (h *K8SModelHandler) PerformClassAction(ctx *RequestContext, action string, query, data *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	man := h.modelManager
	lockKey := fmt.Sprintf("%s-%s", ctx.Cluster().GetName(), man.KeywordPlural())
	lockman.LockClass(ctx.Context(), man, lockKey)
	defer lockman.ReleaseClass(ctx.Context(), man, lockKey)

	specCamel := utils.Kebab2Camel(action, "-")
	modelValue := reflect.ValueOf(man)

	/*if consts.IsRbacEnabled() {
		if err := db.IsObjectRbacAllowed(model, userCred, policy.PolicyActionGet); err != nil {
			return nil, err
		}
	} else if !model.AllowGetDetails(ctx, userCred, query) {
		return nil, httperrors.NewForbiddenError("Not allow to get details")
	}*/
	funcName := fmt.Sprintf("PerformClass%s", specCamel)
	outs, err := callObject(modelValue, funcName, ctx, query, data)
	if err != nil {
		return nil, err
	}
	resVal := outs[0]
	errVal := outs[1].Interface()
	if !gotypes.IsNil(errVal) {
		return nil, errVal.(error)
	}
	if gotypes.IsNil(resVal.Interface()) {
		return nil, nil
	}
	return ValueToJSONObject(resVal), nil
}

func (h *K8SModelHandler) PerformAction(ctx *RequestContext, id, action string, query, data *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	namespace := ctx.GetNamespace()
	model, err := fetchK8SModel(ctx, h.modelManager, namespace, id, query)
	if err != nil {
		return nil, err
	}

	lockman.LockObject(ctx.Context(), model)
	defer lockman.ReleaseObject(ctx.Context(), model)

	specCamel := utils.Kebab2Camel(action, "-")
	modelValue := reflect.ValueOf(model)

	/*if consts.IsRbacEnabled() {
		if err := db.IsObjectRbacAllowed(model, userCred, policy.PolicyActionGet); err != nil {
			return nil, err
		}
	} else if !model.AllowGetDetails(ctx, userCred, query) {
		return nil, httperrors.NewForbiddenError("Not allow to get details")
	}*/
	funcName := fmt.Sprintf("Perform%s", specCamel)
	outs, err := callObject(modelValue, funcName, ctx, query, data)
	if err != nil {
		return nil, err
	}
	resVal := outs[0]
	errVal := outs[1].Interface()
	if !gotypes.IsNil(errVal) {
		return nil, errVal.(error)
	}
	if gotypes.IsNil(resVal.Interface()) {
		return getModelItemDetails(ctx, h.modelManager, model)
	}
	return ValueToJSONObject(resVal), nil
}

func (h *K8SModelHandler) Create(ctx *RequestContext, query, data *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	model, err := DoCreate(h.modelManager, ctx, query, data)
	if err != nil {
		return nil, err
	}
	return getModelItemDetails(ctx, h.modelManager, model)
}

func DoCreate(manager IK8sModelManager, ctx *RequestContext, query, data *jsonutils.JSONDict) (IK8sModel, error) {
	lockKey := fmt.Sprintf("%s-%s", ctx.Cluster().GetId(), manager.KeywordPlural())
	lockman.LockClass(ctx.Context(), manager, lockKey)
	defer lockman.ReleaseClass(ctx.Context(), manager, lockKey)
	model, err := doCreateItem(manager, ctx, query, data)
	return model, err
}

func doCreateItem(
	manager IK8sModelManager,
	ctx *RequestContext,
	query, data *jsonutils.JSONDict) (IK8sModel, error) {
	man := manager
	cluster := ctx.Cluster()
	cli := cluster.GetHandler()
	dataDict, err := ValidateCreateData(man, ctx, query, data)
	if err != nil {
		return nil, k8serrors.NewGeneralError(err)
	}
	version, err := cluster.GetClientset().Discovery().ServerVersion()
	if err != nil {
		return nil, err
	}
	resInfo := man.GetK8sResourceInfo(version)
	obj, err := NewK8SRawObjectForCreate(man, ctx, dataDict)
	if err != nil {
		return nil, k8serrors.NewGeneralError(err)
	}
	obj, err = cli.CreateV2(resInfo.ResourceName, ctx.GetNamespaceByData(), obj)
	if err != nil {
		return nil, k8serrors.NewGeneralError(err)
	}
	return NewK8SModelObject(man, cluster, obj)
}

func (h *K8SModelHandler) Update(ctx *RequestContext, id string, query, data *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	model, err := fetchK8SModel(ctx, h.modelManager, ctx.GetNamespace(), id, query)
	if err != nil {
		return nil, err
	}
	ret, err := DoUpdate(h.modelManager, model, ctx, query, data)
	if err != nil {
		return nil, err
	}
	return getModelItemDetails(ctx, h.modelManager, ret)
}

func DoUpdate(
	manager IK8sModelManager,
	model IK8sModel,
	ctx *RequestContext, query, data *jsonutils.JSONDict) (IK8sModel, error) {
	lockman.LockObject(ctx.Context(), model)
	defer lockman.ReleaseObject(ctx.Context(), model)
	return doUpdateItem(manager, model, ctx, query, data)
}

func doUpdateItem(
	manager IK8sModelManager,
	model IK8sModel,
	ctx *RequestContext, query, data *jsonutils.JSONDict) (IK8sModel, error) {
	data, err := ValidateUpdateData(model, ctx, query, data)
	if err != nil {
		return nil, err
	}
	rawObj, err := NewK8SRawObjectForUpdate(model, ctx, data)
	if err != nil {
		return nil, err
	}
	cli := ctx.Cluster().GetHandler()
	version, err := model.GetCluster().GetClientset().Discovery().ServerVersion()
	if err != nil {
		return nil, err
	}
	resInfo := manager.GetK8sResourceInfo(version)
	_, err = cli.UpdateV2(resInfo.ResourceName, rawObj)
	if err != nil {
		return nil, err
	}
	return NewK8SModelObject(manager, ctx.Cluster(), rawObj)
}

func (h *K8SModelHandler) Delete(ctx *RequestContext, id string, query, data *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	model, err := fetchK8SModel(ctx, h.modelManager, ctx.GetNamespace(), id, query)
	if err != nil {
		return nil, err
	}
	if err := DoDelete(h.modelManager, model, ctx, query, data); err != nil {
		return nil, err
	}
	return getModelItemDetails(ctx, h.modelManager, model)
}

func DoDelete(
	man IK8sModelManager,
	model IK8sModel,
	ctx *RequestContext,
	query, data *jsonutils.JSONDict) error {

	lockman.LockObject(ctx.Context(), model)
	defer lockman.ReleaseObject(ctx.Context(), model)

	if err := ValidateDeleteCondition(model, ctx, query, data); err != nil {
		return err
	}

	if err := CustomizeDelete(model, ctx, query, data); err != nil {
		return err
	}

	meta, err := model.GetObjectMeta()
	if err != nil {
		return err
	}
	cli := ctx.Cluster().GetHandler()
	version, err := model.GetCluster().GetClientset().Discovery().ServerVersion()
	if err != nil {
		return err
	}
	resInfo := man.GetK8sResourceInfo(version)

	if err := cli.Delete(resInfo.ResourceName, meta.GetNamespace(), meta.GetName(), &metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

func (h *K8SModelHandler) GetRawData(ctx *RequestContext, id string, query *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	namespace := ctx.GetNamespaceByQuery()
	model, err := fetchK8SModel(ctx, h.modelManager, namespace, id, query)
	if err != nil {
		return nil, err
	}
	return K8sObjectToJSONObject(model.GetK8sObject()), nil
}

func (h *K8SModelHandler) UpdateRawData(ctx *RequestContext, id string, query, data *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	namespace := ctx.GetNamespaceByQuery()
	model, err := fetchK8SModel(ctx, h.modelManager, namespace, id, query)
	if err != nil {
		return nil, err
	}
	cli := ctx.Cluster().GetHandler()
	version, err := model.GetCluster().GetClientset().Discovery().ServerVersion()
	if err != nil {
		return nil, err
	}
	resInfo := h.modelManager.GetK8sResourceInfo(version)
	rawStr, err := data.GetString()
	if err != nil {
		return nil, httperrors.NewInputParameterError("Get body raw data: %v", err)
	}
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(rawStr), nil, nil)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Decode to runtime object error: %v", err)
	}
	putSpec := runtime.Unknown{}
	objStr, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	if err := json.NewDecoder(strings.NewReader(string(objStr))).Decode(&putSpec); err != nil {
		return nil, err
	}
	_, err = cli.Update(resInfo.ResourceName, model.GetNamespace(), model.GetName(), &putSpec)
	if err != nil {
		return nil, err
	}
	return K8sObjectToJSONObject(obj), nil
}
