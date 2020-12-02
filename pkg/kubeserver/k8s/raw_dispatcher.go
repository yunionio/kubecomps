package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"

	"yunion.io/x/kubecomps/pkg/kubeserver/client"
	clientapi "yunion.io/x/kubecomps/pkg/kubeserver/client/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/resources/common"
	"yunion.io/x/kubecomps/pkg/kubeserver/resources/errors"
)

func AddRawResourceDispatcher(prefix string, app *appsrv.Application) {
	log.Infof("Register k8s raw resource dispatcher")
	clusterPrefix := getClusterPrefix(prefix)

	rawResourcePrefix := fmt.Sprintf("%s/_raw/<kind>/<name>", clusterPrefix)

	// GET raw resource
	app.AddHandler("GET", rawResourcePrefix, auth.Authenticate(getResourceHandler))

	// Get raw resource yaml
	app.AddHandler("GET", fmt.Sprintf("%s/yaml", rawResourcePrefix), auth.Authenticate(getResourceYAMLHandler))

	// PUT raw resource
	app.AddHandler("PUT", rawResourcePrefix, auth.Authenticate(putResourceHandler))

	// DELETE raw resource
	app.AddHandler("DELETE", rawResourcePrefix, auth.Authenticate(deleteResourceHandler))
}

func NewCommonRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) (*common.Request, error) {
	_, query, body := _fetchEnv(ctx, w, r)
	queryDict := jsonutils.NewDict()
	dataDict := jsonutils.NewDict()
	if query != nil {
		queryDict = query.(*jsonutils.JSONDict)
	}
	if body != nil {
		dataDict = body.(*jsonutils.JSONDict)
	}
	return NewCloudK8sRequest(ctx, queryDict, dataDict)
}

type verberEnv struct {
	client      client.ResourceHandler
	kindPlural  string
	namespace   string
	inNamespace bool
	name        string
	request     *common.Request
}

func fetchVerberEnv(ctx context.Context, w http.ResponseWriter, r *http.Request) (*verberEnv, error) {
	req, err := NewCommonRequest(ctx, w, r)
	if err != nil {
		return nil, err
	}
	cli := req.GetVerberClient()
	params := req.GetParams()
	kindPlural := params["<kind>"]
	name := params["<name>"]
	kindPlural = clientapi.TranslateKindPlural(kindPlural)
	resourceSpec, ok := clientapi.KindToResourceMap[kindPlural]
	if !ok {
		return nil, fmt.Errorf("Not found %q resource kind spec", kindPlural)
	}
	inNamespace := resourceSpec.Namespaced
	namespace := ""
	if inNamespace {
		namespace = req.GetDefaultNamespace()
	}
	env := &verberEnv{
		client:      cli,
		kindPlural:  kindPlural,
		inNamespace: inNamespace,
		namespace:   namespace,
		name:        name,
		request:     req,
	}
	return env, nil
}

func (env *verberEnv) Get() (runtime.Object, error) {
	return env.client.Get(env.kindPlural, env.namespace, env.name)
}

func (env *verberEnv) Put() error {
	rawStr, err := env.request.Data.GetString()
	if err != nil {
		return httperrors.NewInputParameterError("Get body string error: %v", err)
	}
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(rawStr), nil, nil)
	if err != nil {
		return httperrors.NewInputParameterError("Decode error: %v", err)
	}
	putSpec := runtime.Unknown{}
	objStr, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	if err := json.NewDecoder(strings.NewReader(string(objStr))).Decode(&putSpec); err != nil {
		return err
	}
	log.Debugf("Input %s, get object: %#v", rawStr, putSpec)
	_, err = env.client.Update(env.kindPlural, env.namespace, env.name, &putSpec)
	return err
}

func (env *verberEnv) Delete() error {
	return env.client.Delete(env.kindPlural, env.namespace, env.name, &metav1.DeleteOptions{})
}

func getResourceHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	env, err := fetchVerberEnv(ctx, w, r)
	if err != nil {
		errors.GeneralServerError(ctx, w, err)
		return
	}
	obj, err := env.Get()
	if err != nil {
		errors.GeneralServerError(ctx, w, err)
		return
	}
	SendJSON(w, obj)
}

func getResourceYAMLHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	env, err := fetchVerberEnv(ctx, w, r)
	if err != nil {
		errors.GeneralServerError(ctx, w, err)
		return
	}
	obj, err := env.Get()
	if err != nil {
		errors.GeneralServerError(ctx, w, err)
		return
	}
	SendYAML(w, obj)
}

func putResourceHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	env, err := fetchVerberEnv(ctx, w, r)
	if err != nil {
		errors.GeneralServerError(ctx, w, err)
		return
	}
	err = env.Put()
	if err != nil {
		errors.GeneralServerError(ctx, w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func deleteResourceHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	env, err := fetchVerberEnv(ctx, w, r)
	if err != nil {
		errors.GeneralServerError(ctx, w, err)
		return
	}
	err = env.Delete()
	if err != nil {
		errors.GeneralServerError(ctx, w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}
