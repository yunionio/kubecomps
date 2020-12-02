package k8s

import (
	"context"
	"fmt"
	"reflect"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/utils"

	//"yunion.io/x/kubecomps/pkg/kubeserver/models/types"
	"yunion.io/x/kubecomps/pkg/kubeserver/client"
	clientapi "yunion.io/x/kubecomps/pkg/kubeserver/client/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/resources/common"
	"yunion.io/x/kubecomps/pkg/kubeserver/resources/errors"
)

type IK8sResourceHandler interface {
	dispatcher.IMiddlewareFilter

	Keyword() string
	KeywordPlural() string

	List(ctx context.Context, query *jsonutils.JSONDict) (common.ListResource, error)

	Get(ctx context.Context, id string, query *jsonutils.JSONDict) (interface{}, error)

	GetSpecific(ctx context.Context, id, spec string, query *jsonutils.JSONDict) (interface{}, error)

	Create(ctx context.Context, query *jsonutils.JSONDict, data *jsonutils.JSONDict) (interface{}, error)

	PerformClassAction(ctx context.Context, action string, query, data *jsonutils.JSONDict) (interface{}, error)

	PerformAction(ctx context.Context, id, action string, query, data *jsonutils.JSONDict) (interface{}, error)

	Update(ctx context.Context, id string, query *jsonutils.JSONDict, data *jsonutils.JSONDict) (interface{}, error)

	Delete(ctx context.Context, id string, query *jsonutils.JSONDict, data *jsonutils.JSONDict) error
}

type IK8sResourceManager interface {
	Keyword() string
	KeywordPlural() string

	InNamespace() bool

	// list hooks
	AllowListItems(req *common.Request) bool
	List(req *common.Request) (common.ListResource, error)

	// get hooks
	AllowGetItem(req *common.Request, id string) bool
	Get(req *common.Request, id string) (interface{}, error)

	// create hooks
	AllowCreateItem(req *common.Request) bool
	ValidateCreateData(req *common.Request) error
	Create(req *common.Request) (interface{}, error)

	// update hooks
	AllowUpdateItem(req *common.Request, id string) bool
	Update(req *common.Request, id string) (interface{}, error)

	// delete hooks
	AllowDeleteItem(req *common.Request, id string) bool
	Delete(req *common.Request, id string) error

	IsRawResource() bool
}

type K8sResourceHandler struct {
	resourceManager IK8sResourceManager
}

func NewK8sResourceHandler(man IK8sResourceManager) *K8sResourceHandler {
	return &K8sResourceHandler{man}
}

func (h *K8sResourceHandler) Filter(f appsrv.FilterHandler) appsrv.FilterHandler {
	return auth.Authenticate(f)
}

func (h *K8sResourceHandler) Keyword() string {
	return h.resourceManager.Keyword()
}

func (h *K8sResourceHandler) KeywordPlural() string {
	return h.resourceManager.KeywordPlural()
}

func getUserCredential(ctx context.Context) mcclient.TokenCredential {
	return policy.FetchUserCredential(ctx)
}

func getCluster(query, data *jsonutils.JSONDict, userCred mcclient.TokenCredential) (*models.SCluster, error) {
	var clusterId string
	for _, src := range []*jsonutils.JSONDict{query, data} {
		if src == nil {
			continue
		}
		clusterId, _ = src.GetString("cluster")
		if clusterId != "" {
			break
		}
	}
	if clusterId == "" {
		return nil, httperrors.NewMissingParameterError("cluster")
	}
	cluster, err := models.ClusterManager.FetchClusterByIdOrName(userCred, clusterId)
	if err != nil {
		return nil, err
	}
	return cluster.(*models.SCluster), nil
}

func newK8sAdminClient(cluster *models.SCluster) (kubernetes.Interface, *rest.Config, error) {
	cli, err := cluster.GetK8sClient()
	if err != nil {
		return nil, nil, err
	}
	config, err := cluster.GetK8sRestConfig()
	if err != nil {
		return nil, nil, err
	}
	return cli, config, err
}

func NewCloudK8sRequest(ctx context.Context, query, data *jsonutils.JSONDict) (*common.Request, error) {
	userCred := getUserCredential(ctx)

	cluster, err := getCluster(query, data, userCred)
	if err != nil {
		return nil, err
	}

	/*k8sCli, config, err := newK8sUserClient(cluster, userCred)
	if err != nil {
		return nil, err
	}*/
	man, err := client.GetManagerByCluster(cluster)
	if err != nil {
		return nil, err
	}

	k8sAdminCli, adminConfig, err := newK8sAdminClient(cluster)
	if err != nil {
		return nil, err
	}
	kubeAdminConfig, err := cluster.GetAdminKubeconfig()
	if err != nil {
		return nil, err
	}
	req := &common.Request{
		Cluster:        cluster,
		ClusterManager: man,
		//K8sClient:       k8sCli,
		//K8sConfig:       config,
		K8sAdminClient:  k8sAdminCli,
		K8sAdminConfig:  adminConfig,
		UserCred:        userCred,
		Query:           query,
		Data:            data,
		Context:         ctx,
		KubeAdminConfig: kubeAdminConfig,
	}
	//if err := req.EnsureProjectNamespaces(); err != nil {
	//return req, err
	//}
	return req, nil
}

func (h *K8sResourceHandler) List(ctx context.Context, query *jsonutils.JSONDict) (common.ListResource, error) {
	req, err := NewCloudK8sRequest(ctx, query, nil)
	if err != nil {
		return nil, errors.NewJSONClientError(err)
	}
	if !h.resourceManager.AllowListItems(req) {
		return nil, httperrors.NewForbiddenError("Not allow to list")
	}
	items, err := listItems(h.resourceManager, req)
	if err != nil {
		log.Errorf("Fail to list items: %v", err)
		return nil, httperrors.NewGeneralError(err)
	}
	return items, nil
}

func listItems(
	man IK8sResourceManager,
	req *common.Request,
) (common.ListResource, error) {
	ret, err := man.List(req)
	return ret, err
}

func (h *K8sResourceHandler) Get(ctx context.Context, id string, query *jsonutils.JSONDict) (interface{}, error) {
	req, err := NewCloudK8sRequest(ctx, query, nil)
	if err != nil {
		return nil, errors.NewJSONClientError(err)
	}
	if !h.resourceManager.AllowGetItem(req, id) {
		return nil, httperrors.NewForbiddenError("Not allow to get item")
	}
	return h.resourceManager.Get(req, id)
}

func (h *K8sResourceHandler) GetSpecific(ctx context.Context, id, spec string, query *jsonutils.JSONDict) (interface{}, error) {
	req, err := NewCloudK8sRequest(ctx, query, nil)
	if err != nil {
		return nil, errors.NewJSONClientError(err)
	}
	specCamel := utils.Kebab2Camel(spec, "-")
	funcName := fmt.Sprintf("AllowGetDetails%s", specCamel)
	funcValue, err := getManagerFuncValue(funcName, h.resourceManager)
	if err != nil {
		return nil, httperrors.NewSpecNotFoundError("%s", err.Error())
	}
	params := []reflect.Value{
		reflect.ValueOf(req),
		reflect.ValueOf(id),
	}
	outs := funcValue.Call(params)
	if len(outs) != 1 {
		return nil, httperrors.NewInternalServerError("Invalid %s return value", funcName)
	}
	isAllow := outs[0].Bool()
	if !isAllow {
		return nil, httperrors.NewForbiddenError("%s not allow to get spec %s", h.Keyword(), spec)
	}

	funcName = fmt.Sprintf("GetDetails%s", specCamel)
	funcValue, err = getManagerFuncValue(funcName, h.resourceManager)
	if err != nil {
		return nil, httperrors.NewSpecNotFoundError("%s", err.Error())
	}

	outs = funcValue.Call(params)
	if len(outs) != 2 {
		return nil, httperrors.NewInternalServerError("Invalid %s return value", funcName)
	}

	resVal := outs[0].Interface()
	errVal := outs[1].Interface()
	if !gotypes.IsNil(errVal) {
		return nil, errVal.(error)
	}
	if gotypes.IsNil(resVal) {
		return nil, nil
	}
	return resVal, nil
}

func (h *K8sResourceHandler) PerformClassAction(ctx context.Context, action string, query, data *jsonutils.JSONDict) (interface{}, error) {
	req, err := NewCloudK8sRequest(ctx, query, data)
	if err != nil {
		return nil, errors.NewJSONClientError(err)
	}
	specCamel := utils.Kebab2Camel(action, "-")
	funcName := fmt.Sprintf("PerformClass%s", specCamel)
	funcValue, err := getManagerFuncValue(funcName, h.resourceManager)
	if err != nil {
		return nil, httperrors.NewActionNotFoundError("%s", err.Error())
	}
	params := []reflect.Value{
		reflect.ValueOf(req),
	}
	allowFuncName := fmt.Sprintf("Allow%s", funcName)
	allowFuncValue, err := getManagerFuncValue(allowFuncName, h.resourceManager)
	if err != nil {
		return nil, httperrors.NewActionNotFoundError("%s", err.Error())
	}
	outs := allowFuncValue.Call(params)
	if len(outs) != 1 {
		return nil, httperrors.NewInternalServerError("Invalid %s return value", allowFuncName)
	}
	isAllow := outs[0].Bool()
	if !isAllow {
		return nil, httperrors.NewForbiddenError("%s not allow to perform action %s", h.Keyword(), action)
	}
	outs = funcValue.Call(params)
	if len(outs) != 2 {
		return nil, httperrors.NewInternalServerError("Invalid %s return value", funcName)
	}

	resVal := outs[0].Interface()
	errVal := outs[1].Interface()
	if !gotypes.IsNil(errVal) {
		return nil, errVal.(error)
	}
	if gotypes.IsNil(resVal) {
		return nil, nil
	}
	return resVal, nil
}

func (h *K8sResourceHandler) PerformAction(ctx context.Context, id, action string, query, data *jsonutils.JSONDict) (interface{}, error) {
	req, err := NewCloudK8sRequest(ctx, query, data)
	if err != nil {
		return nil, errors.NewJSONClientError(err)
	}
	specCamel := utils.Kebab2Camel(action, "-")
	funcName := fmt.Sprintf("Perform%s", specCamel)
	funcValue, err := getManagerFuncValue(funcName, h.resourceManager)
	if err != nil {
		return nil, httperrors.NewActionNotFoundError("%s", err.Error())
	}
	params := []reflect.Value{
		reflect.ValueOf(req),
		reflect.ValueOf(id),
	}
	allowFuncName := fmt.Sprintf("Allow%s", funcName)
	allowFuncValue, err := getManagerFuncValue(allowFuncName, h.resourceManager)
	if err != nil {
		return nil, httperrors.NewActionNotFoundError("%s", err.Error())
	}
	outs := allowFuncValue.Call(params)
	if len(outs) != 1 {
		return nil, httperrors.NewInternalServerError("Invalid %s return value", allowFuncName)
	}
	isAllow := outs[0].Bool()
	if !isAllow {
		return nil, httperrors.NewForbiddenError("%s not allow to perform action %s", h.Keyword(), action)
	}
	outs = funcValue.Call(params)
	if len(outs) != 2 {
		return nil, httperrors.NewInternalServerError("Invalid %s return value", funcName)
	}

	resVal := outs[0].Interface()
	errVal := outs[1].Interface()
	if !gotypes.IsNil(errVal) {
		return nil, errVal.(error)
	}
	if gotypes.IsNil(resVal) {
		return nil, nil
	}
	return resVal, nil
}

func getManagerFuncValue(funcName string, man IK8sResourceManager) (reflect.Value, error) {
	manValue := reflect.ValueOf(man)
	funcValue := manValue.MethodByName(funcName)
	if !funcValue.IsValid() || funcValue.IsNil() {
		return reflect.ValueOf(nil), fmt.Errorf("Not found function %s on %s manager", funcName, man.Keyword())
	}
	return funcValue, nil
}

func doCreateItem(man IK8sResourceManager, req *common.Request) (jsonutils.JSONObject, error) {
	err := man.ValidateCreateData(req)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	res, err := man.Create(req)
	if err != nil {
		log.Errorf("Fail to create resource: %v", err)
		return nil, err
	}
	return jsonutils.Marshal(res), nil
}

func (h *K8sResourceHandler) Create(ctx context.Context, query, data *jsonutils.JSONDict) (interface{}, error) {
	req, err := NewCloudK8sRequest(ctx, query, data)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if !h.resourceManager.AllowCreateItem(req) {
		return nil, httperrors.NewForbiddenError("Not allow to create item")
	}

	return doCreateItem(h.resourceManager, req)
}

func (h *K8sResourceHandler) Update(ctx context.Context, id string, query, data *jsonutils.JSONDict) (interface{}, error) {
	req, err := NewCloudK8sRequest(ctx, query, data)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	obj, err := doUpdateItem(h.resourceManager, req, id)

	if err != nil {
		return nil, errors.NewJSONClientError(err)
	}
	return obj, err
}

func doUpdateItem(man IK8sResourceManager, req *common.Request, id string) (interface{}, error) {
	if !man.AllowUpdateItem(req, id) {
		return nil, httperrors.NewForbiddenError("Not allow to delete")
	}
	return man.Update(req, id)
}

func (h *K8sResourceHandler) Delete(ctx context.Context, id string, query, data *jsonutils.JSONDict) error {
	req, err := NewCloudK8sRequest(ctx, query, data)
	if err != nil {
		return httperrors.NewGeneralError(err)
	}

	if h.resourceManager.IsRawResource() {
		err = doRawDelete(h.resourceManager, req, id)
	} else {
		err = doDeleteItem(h.resourceManager, req, id)
	}

	if err != nil {
		return errors.NewJSONClientError(err)
	}
	return nil
}

func doRawDelete(man IK8sResourceManager, req *common.Request, id string) error {
	verber := req.GetVerberClient()

	kindPlural := clientapi.TranslateKindPlural(man.KeywordPlural())
	namespace := ""
	inNamespace := man.InNamespace()
	if inNamespace {
		namespace = req.GetDefaultNamespace()
	}
	if err := verber.Delete(kindPlural, namespace, id, &metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

func doDeleteItem(man IK8sResourceManager, req *common.Request, id string) error {
	if !man.AllowDeleteItem(req, id) {
		return httperrors.NewForbiddenError("Not allow to delete")
	}
	return man.Delete(req, id)
}
