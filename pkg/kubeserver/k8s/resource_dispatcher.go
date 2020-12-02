package k8s

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"

	"yunion.io/x/kubecomps/pkg/kubeserver/resources/common"
	"yunion.io/x/kubecomps/pkg/kubeserver/resources/errors"
)

func getClusterPrefix(prefix string) string {
	return prefix
}

func AddResourceDispatcher(prefix string, app *appsrv.Application, handler IK8sResourceHandler) {
	log.Infof("Register k8s resource %s", handler.Keyword())

	clusterPrefix := getClusterPrefix(prefix)
	plural := handler.KeywordPlural()
	metadata := map[string]interface{}{"manager": handler}
	tags := map[string]string{"resource": plural}

	// list resources
	app.AddHandler2("GET", fmt.Sprintf("%s/%s", clusterPrefix, plural), handler.Filter(listHandler), metadata, "list", tags)

	// get resource instance details
	app.AddHandler2("GET", fmt.Sprintf("%s/%s/<resid>", clusterPrefix, plural), handler.Filter(getHandler), metadata, "get_details", tags)

	// get resource spec
	app.AddHandler2("GET", fmt.Sprintf("%s/%s/<resid>/<spec>", clusterPrefix, plural), handler.Filter(getSpecHandler), metadata, "get_specific", tags)

	// create resources
	app.AddHandler2("POST", fmt.Sprintf("%s/%s", clusterPrefix, plural), handler.Filter(createHandler), metadata, "create", tags)

	// perform action on resource manager
	app.AddHandler2("POST", fmt.Sprintf("%s/%s/<action>", clusterPrefix, plural), handler.Filter(performClassActionHandler), metadata, "perform_class_action", tags)

	// perform action on resource instance
	app.AddHandler2("POST", fmt.Sprintf("%s/%s/<resid>/<action>", clusterPrefix, plural), handler.Filter(performActionHandler), metadata, "perform_action", tags)

	// update resource
	app.AddHandler2("PUT", fmt.Sprintf("%s/%s/<resid>", clusterPrefix, plural), handler.Filter(updateHandler), metadata, "update", tags)

	// delete resource
	app.AddHandler2("DELETE", fmt.Sprintf("%s/%s/<resid>", clusterPrefix, plural), handler.Filter(deleteHandler), metadata, "delete", tags)
}

func _fetchEnv(ctx context.Context, w http.ResponseWriter, r *http.Request) (map[string]string, jsonutils.JSONObject, jsonutils.JSONObject) {
	// trace := appsrv.AppContextTrace(ctx)
	params := appctx.AppContextParams(ctx)
	query, e := jsonutils.ParseQueryString(r.URL.RawQuery)
	if e != nil {
		log.Errorf("Parse query string %s failed: %s", r.URL.RawQuery, e)
	}
	var body jsonutils.JSONObject = nil
	if r.Method == "PUT" || r.Method == "POST" || r.Method == "DELETE" || r.Method == "PATCH" {
		body, e = appsrv.FetchJSON(r)
		if e != nil {
			log.Errorf("Fail to decode JSON request body: %s", e)
		}
	}
	return params, query, body
}

func fetchEnv(ctx context.Context, w http.ResponseWriter, r *http.Request) (
	IK8sResourceHandler, map[string]string, jsonutils.JSONObject, jsonutils.JSONObject) {
	params, query, body := _fetchEnv(ctx, w, r)
	metadata := appctx.AppContextMetadata(ctx)
	handler, ok := metadata["manager"].(IK8sResourceHandler)
	if !ok {
		log.Fatalf("No manager found for URL: %s", r.URL)
	}
	return handler, params, query, body
}

func listHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	handler, _, query, _ := fetchEnv(ctx, w, r)
	handleList(ctx, w, handler, query)
}

func handleList(ctx context.Context, w http.ResponseWriter, handler IK8sResourceHandler, query jsonutils.JSONObject) {
	result, err := handler.List(ctx, query.(*jsonutils.JSONDict))
	if err != nil {
		errors.GeneralServerError(ctx, w, err)
		return
	}
	SendJSON(w, common.ListResource2JSONWithKey(result, handler.KeywordPlural()))
}

func wrapBody(body interface{}, key string) map[string]interface{} {
	ret := make(map[string]interface{})
	if body != nil {
		ret[key] = body
	}
	return ret
}

func wrapJBody(body jsonutils.JSONObject, key string) jsonutils.JSONObject {
	if body == nil {
		return nil
	}
	ret := jsonutils.NewDict()
	ret.Add(body, key)
	return ret
}

func getHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	handler, params, query, _ := fetchEnv(ctx, w, r)
	result, err := handler.Get(ctx, params["<resid>"], query.(*jsonutils.JSONDict))
	if err != nil {
		errors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendJSON(w, wrapJBody(jsonutils.Marshal(result), handler.Keyword()))
}

func getSpecHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	handler, params, query, _ := fetchEnv(ctx, w, r)
	result, err := handler.GetSpecific(ctx, params["<resid>"], params["<spec>"], query.(*jsonutils.JSONDict))
	if err != nil {
		errors.GeneralServerError(ctx, w, err)
		return
	}
	SendJSON(w, wrapBody(result, handler.Keyword()))
}

func createHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	handler, _, query, body := fetchEnv(ctx, w, r)
	handleCreate(ctx, w, handler, query, body)
}

func performClassActionHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	handler, params, query, body := fetchEnv(ctx, w, r)
	var data jsonutils.JSONObject
	if body != nil {
		if body.Contains(handler.KeywordPlural()) {
			data, _ = body.Get(handler.KeywordPlural())
		} else {
			data = body
		}
	} else {
		data = jsonutils.NewDict()
	}
	result, err := handler.PerformClassAction(ctx, params["<action>"], query.(*jsonutils.JSONDict), data.(*jsonutils.JSONDict))
	if err != nil {
		errors.GeneralServerError(ctx, w, err)
		return
	}
	SendJSON(w, wrapBody(result, handler.KeywordPlural()))
}

func performActionHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	handler, params, query, body := fetchEnv(ctx, w, r)
	var data jsonutils.JSONObject
	if body != nil {
		if body.Contains(handler.Keyword()) {
			data, _ = body.Get(handler.Keyword())
		} else {
			data = body
		}
	} else {
		data = jsonutils.NewDict()
	}
	result, err := handler.PerformAction(ctx, params["<resid>"], params["<action>"], query.(*jsonutils.JSONDict), data.(*jsonutils.JSONDict))
	if err != nil {
		errors.GeneralServerError(ctx, w, err)
		return
	}
	SendJSON(w, wrapBody(result, handler.Keyword()))
}

func handleCreate(ctx context.Context, w http.ResponseWriter, handler IK8sResourceHandler, query, body jsonutils.JSONObject) {
	data, err := body.Get(handler.Keyword())
	if err != nil {
		httperrors.InvalidInputError(ctx, w, fmt.Sprintf("No request key: %s", handler.Keyword()))
		return
	}
	result, err := handler.Create(ctx, query.(*jsonutils.JSONDict), data.(*jsonutils.JSONDict))
	if err != nil {
		errors.GeneralServerError(ctx, w, err)
		return
	}
	SendJSON(w, wrapBody(result, handler.Keyword()))
}

func updateHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	handler, params, query, body := fetchEnv(ctx, w, r)
	data, err := body.Get(handler.Keyword())
	if err != nil {
		httperrors.InvalidInputError(ctx, w, fmt.Sprintf("No Request key: %s", handler.Keyword()))
		return
	}
	result, err := handler.Update(ctx, params["<resid>"], query.(*jsonutils.JSONDict), data.(*jsonutils.JSONDict))
	if err != nil {
		errors.GeneralServerError(ctx, w, err)
		return
	}
	SendJSON(w, wrapBody(result, handler.Keyword()))
}

func deleteHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	handler, params, query, body := fetchEnv(ctx, w, r)
	var data jsonutils.JSONObject
	var err error
	if body != nil {
		data, err = body.Get(handler.Keyword())
		if err != nil {
			httperrors.InvalidInputError(ctx, w, fmt.Sprintf("No request key: %s", handler.Keyword()))
			return
		}
	}

	q := jsonutils.NewDict()
	if query != nil {
		q = query.(*jsonutils.JSONDict)
	}
	d := jsonutils.NewDict()
	if data != nil {
		d = data.(*jsonutils.JSONDict)
	}
	err = handler.Delete(ctx, params["<resid>"], q, d)
	if err != nil {
		errors.GeneralServerError(ctx, w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}
