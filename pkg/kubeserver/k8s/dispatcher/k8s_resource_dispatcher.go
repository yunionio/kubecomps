package dispatcher

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/pkg/appctx"
	"yunion.io/x/pkg/util/printutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/kubecomps/pkg/kubeserver/client"
	"yunion.io/x/kubecomps/pkg/kubeserver/k8s/common/model"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	"yunion.io/x/kubecomps/pkg/utils/k8serrors"
)

type IK8sModelDispatchHandler interface {
	dispatcher.IMiddlewareFilter

	Keyword() string
	KeywordPlural() string
	//ContextKeywordPlurals() [][]string

	//List(ctx context.Context, query *jsonutils.JSONDict, ctxIds []dispatcher.SResourceContext) (*modulebase.ListResult, error)
	List(ctx *model.RequestContext, query *jsonutils.JSONDict) (*printutils.ListResult, error)
	Get(ctx *model.RequestContext, id string, query *jsonutils.JSONDict) (jsonutils.JSONObject, error)
	GetSpecific(ctx *model.RequestContext, id string, spec string, query *jsonutils.JSONDict) (jsonutils.JSONObject, error)
	Create(ctx *model.RequestContext, query, data *jsonutils.JSONDict) (jsonutils.JSONObject, error)
	PerformClassAction(ctx *model.RequestContext, action string, query, data *jsonutils.JSONDict) (jsonutils.JSONObject, error)
	PerformAction(ctx *model.RequestContext, id string, action string, query, data *jsonutils.JSONDict) (jsonutils.JSONObject, error)
	Update(ctx *model.RequestContext, id string, query, data *jsonutils.JSONDict) (jsonutils.JSONObject, error)
	Delete(ctx *model.RequestContext, id string, query, data *jsonutils.JSONDict) (jsonutils.JSONObject, error)
	GetRawData(ctx *model.RequestContext, id string, query *jsonutils.JSONDict) (jsonutils.JSONObject, error)
	UpdateRawData(ctx *model.RequestContext, id string, query, data *jsonutils.JSONDict) (jsonutils.JSONObject, error)
}

func getClusterPrefix(prefix string) string {
	// return path.Join(prefix, "")
	return prefix
}

type K8sModelDispatcher struct {
	prefix string
	app    *appsrv.Application
}

func NewK8sModelDispatcher(prefix string, app *appsrv.Application) *K8sModelDispatcher {
	return &K8sModelDispatcher{
		prefix: prefix,
		app:    app,
	}
}

func (d K8sModelDispatcher) Add(handler IK8sModelDispatchHandler) {
	prefix := d.prefix
	app := d.app

	clusterPrefix := getClusterPrefix(prefix)
	plural := handler.KeywordPlural()
	metadata := map[string]interface{}{"manager": handler}
	tags := map[string]string{"resource": plural}

	// list
	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s", clusterPrefix, plural),
		handler.Filter(d.list), metadata, "list", tags)
	// get
	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/<resid>", clusterPrefix, handler.KeywordPlural()),
		handler.Filter(d.get), metadata, "get_details", tags)
	// get specific
	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/<resid>/<spec>", clusterPrefix, handler.KeywordPlural()),
		handler.Filter(d.getSpec), metadata, "get_specific", tags)

	// create
	app.AddHandler2("POST",
		fmt.Sprintf("%s/%s", clusterPrefix, handler.KeywordPlural()),
		handler.Filter(d.create), metadata, "create", tags)
	// perform action on resource manager
	app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/<action>", clusterPrefix, handler.KeywordPlural()),
		handler.Filter(d.performClassAction), metadata, "perform_class_action", tags)
	// perform action on resource instance
	app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/<resid>/<action>", clusterPrefix, handler.KeywordPlural()),
		handler.Filter(d.performAction), metadata, "perform_action", tags)

	// update
	app.AddHandler2("PUT",
		fmt.Sprintf("%s/%s/<resid>", clusterPrefix, handler.KeywordPlural()),
		handler.Filter(d.update), metadata, "update", tags)

	// delete
	app.AddHandler2("DELETE",
		fmt.Sprintf("%s/%s/<resid>", clusterPrefix, handler.KeywordPlural()),
		handler.Filter(d.delete), metadata, "delete", tags)

	// raw data dispatch
	// get k8s object raw data
	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/<resid>/rawdata", clusterPrefix, handler.KeywordPlural()),
		handler.Filter(d.getRawData), metadata, "get_raw_data", tags)
	// update k8s object by raw data
	app.AddHandler2("PUT",
		fmt.Sprintf("%s/%s/<resid>/rawdata", clusterPrefix, handler.KeywordPlural()),
		handler.Filter(d.updateRawData), metadata, "get_raw_data", tags)
}

func (d K8sModelDispatcher) fetchEnv(
	ctx context.Context, w http.ResponseWriter, r *http.Request) (
	IK8sModelDispatchHandler, map[string]string,
	*jsonutils.JSONDict, *jsonutils.JSONDict) {
	params, query, body := appsrv.FetchEnv(ctx, w, r)
	metadata := appctx.AppContextMetadata(ctx)
	handler, ok := metadata["manager"].(IK8sModelDispatchHandler)
	if !ok {
		log.Fatalf("No manager found for URL: %s", r.URL)
	}
	qDict := jsonutils.NewDict()
	dDict := jsonutils.NewDict()
	if query != nil {
		qDict = query.(*jsonutils.JSONDict)
	}
	if body != nil {
		dDict = body.(*jsonutils.JSONDict)
	}
	return handler, params, qDict, dDict
}

func mergeQueryParams(params map[string]string, query jsonutils.JSONObject, excludes ...string) *jsonutils.JSONDict {
	if query == nil {
		query = jsonutils.NewDict()
	}
	queryDict := query.(*jsonutils.JSONDict)
	for k, v := range params {
		if !utils.IsInStringArray(k, excludes) {
			queryDict.Add(jsonutils.NewString(v), k[1:len(k)-1])
		}
	}
	return queryDict
}

func (d K8sModelDispatcher) getCluster(ctx context.Context, query, data *jsonutils.JSONDict, userCred mcclient.TokenCredential) (*client.ClusterManager, error) {
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
	cluster, err := models.ClusterManager.FetchClusterByIdOrName(ctx, userCred, clusterId)
	if err != nil {
		return nil, err
	}
	return client.GetManagerByCluster(cluster)
}

func getUserCredential(ctx context.Context) mcclient.TokenCredential {
	return policy.FetchUserCredential(ctx)
}

func (d K8sModelDispatcher) getContext(ctx context.Context, query, data *jsonutils.JSONDict) (*model.RequestContext, error) {
	userCred := getUserCredential(ctx)
	cluster, err := d.getCluster(ctx, query, data, userCred)
	if err != nil {
		return nil, err
	}
	return model.NewRequestContext(ctx, userCred, cluster, query, data), nil
}

func wrapBody(body jsonutils.JSONObject, key string) jsonutils.JSONObject {
	if body == nil {
		return nil
	}
	ret := jsonutils.NewDict()
	ret.Add(body, key)
	return ret
}

func (d K8sModelDispatcher) fetchContextParams(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	excludes ...string,
) (IK8sModelDispatchHandler, *model.RequestContext, map[string]string, error) {
	handler, params, query, body := d.fetchEnv(ctx, w, r)
	if query != nil {
		query = mergeQueryParams(params, query, excludes...)
	}
	data := jsonutils.NewDict()
	if body != nil {
		if body.Contains(handler.Keyword()) {
			tmpData, _ := body.Get(handler.Keyword())
			if tmpData == nil {
				data = body
			} else {
				data = tmpData.(*jsonutils.JSONDict)
			}
		} else if body.Contains(handler.KeywordPlural()) {
			tmpData, _ := body.Get(handler.KeywordPlural())
			if tmpData == nil {
				data = body
			} else {
				data = tmpData.(*jsonutils.JSONDict)
			}
		} else {
			data = body
		}
	}
	reqCtx, err := d.getContext(ctx, query, data)
	if err != nil {
		return nil, nil, nil, err
	}
	return handler, reqCtx, params, nil
}

func (d K8sModelDispatcher) list(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	handler, reqCtx, _, err := d.fetchContextParams(ctx, w, r)
	if err != nil {
		k8serrors.GeneralServerError(ctx, w, err)
		return
	}
	result, err := handler.List(reqCtx, reqCtx.GetQuery())
	if err != nil {
		k8serrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendJSON(w, modulebase.ListResult2JSONWithKey(result, handler.KeywordPlural()))
}

func (d K8sModelDispatcher) get(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	handler, reqCtx, params, err := d.fetchContextParams(ctx, w, r, "<resid>")
	if err != nil {
		k8serrors.GeneralServerError(ctx, w, err)
		return
	}
	result, err := handler.Get(reqCtx, params["<resid>"], reqCtx.GetQuery())
	if err != nil {
		k8serrors.GeneralServerError(ctx, w, err)
		return
	}
	// log.Errorf("result string: %s", result.String())
	// k8s.SendJSON(w, result)
	appsrv.SendJSON(w, wrapBody(result, handler.Keyword()))
}

func (d K8sModelDispatcher) getSpec(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	handler, reqCtx, params, err := d.fetchContextParams(ctx, w, r, "<resid>", "<spec>")
	if err != nil {
		k8serrors.GeneralServerError(ctx, w, err)
		return
	}
	result, err := handler.GetSpecific(reqCtx, params["<resid>"], params["<spec>"], reqCtx.GetQuery())
	if err != nil {
		k8serrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendJSON(w, wrapBody(result, handler.Keyword()))
}

func (d K8sModelDispatcher) create(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	handler, reqCtx, _, err := d.fetchContextParams(ctx, w, r)
	if err != nil {
		k8serrors.GeneralServerError(ctx, w, err)
		return
	}
	result, err := handler.Create(reqCtx, reqCtx.GetQuery(), reqCtx.GetData())
	if err != nil {
		k8serrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendJSON(w, wrapBody(result, handler.Keyword()))
}

func (d K8sModelDispatcher) performClassAction(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	handler, reqCtx, params, err := d.fetchContextParams(ctx, w, r, "<action>")
	if err != nil {
		k8serrors.GeneralServerError(ctx, w, err)
		return
	}
	result, err := handler.PerformClassAction(reqCtx, params["<action>"], reqCtx.GetQuery(), reqCtx.GetData())
	if err != nil {
		k8serrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendJSON(w, wrapBody(result, handler.KeywordPlural()))
}

func (d K8sModelDispatcher) performAction(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	handler, reqCtx, params, err := d.fetchContextParams(ctx, w, r, "<resid>", "<action>")
	if err != nil {
		k8serrors.GeneralServerError(ctx, w, err)
		return
	}
	result, err := handler.PerformAction(reqCtx, params["<resid>"], params["<action>"], reqCtx.GetQuery(), reqCtx.GetData())
	if err != nil {
		k8serrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendJSON(w, wrapBody(result, handler.Keyword()))
}

func (d K8sModelDispatcher) update(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	handler, reqCtx, params, err := d.fetchContextParams(ctx, w, r, "<resid>")
	if err != nil {
		k8serrors.GeneralServerError(ctx, w, err)
		return
	}
	result, err := handler.Update(reqCtx, params["<resid>"], reqCtx.GetQuery(), reqCtx.GetData())
	if err != nil {
		k8serrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendJSON(w, wrapBody(result, handler.Keyword()))
}

func (d K8sModelDispatcher) delete(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	handler, reqCtx, params, err := d.fetchContextParams(ctx, w, r, "<resid>")
	if err != nil {
		k8serrors.GeneralServerError(ctx, w, err)
		return
	}
	result, err := handler.Delete(reqCtx, params["<resid>"], reqCtx.GetQuery(), reqCtx.GetData())
	if err != nil {
		k8serrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendJSON(w, wrapBody(result, handler.Keyword()))
}

func (d K8sModelDispatcher) getRawData(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	handler, reqCtx, params, err := d.fetchContextParams(ctx, w, r, "<resid>")
	if err != nil {
		k8serrors.GeneralServerError(ctx, w, err)
		return
	}
	result, err := handler.GetRawData(reqCtx, params["<resid>"], reqCtx.GetQuery())
	if err != nil {
		k8serrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendJSON(w, wrapBody(result, handler.Keyword()))
}

func (d K8sModelDispatcher) updateRawData(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	handler, reqCtx, params, err := d.fetchContextParams(ctx, w, r, "<resid>")
	if err != nil {
		k8serrors.GeneralServerError(ctx, w, err)
		return
	}
	result, err := handler.UpdateRawData(reqCtx, params["<resid>"], reqCtx.GetQuery(), reqCtx.GetData())
	if err != nil {
		k8serrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendJSON(w, wrapBody(result, handler.Keyword()))
}
