package k8s

import (
	"context"
	"fmt"
	"net/http"
	//"net/url"

	//"k8s.io/api/core/v1"
	//"k8s.io/client-go/kubernetes/scheme"
	//"k8s.io/client-go/tools/remotecommand"
	//"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	//"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	//"yunion.io/x/kubecomps/pkg/kubeserver/clusterrouter/proxy"
)

func AddMiscDispatcher(prefix string, app *appsrv.Application) {
	log.Infof("Register k8s misc dispatcher handler")
	clusterPrefix := getClusterPrefix(prefix)

	// handle exec shell
	app.AddHandler("GET",
		fmt.Sprintf("%s/pods/<pod>/shell/<container>", clusterPrefix),
		auth.Authenticate(handleExecShell))
}

func handleExecShell(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	httperrors.GeneralServerError(ctx, w, fmt.Errorf("Not impl"))
	return
	/*params, query, data := _fetchEnv(ctx, w, r)
	request, err := NewCloudK8sRequest(ctx, query.(*jsonutils.JSONDict), nil)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	cluster, _ := getCluster(query.(*jsonutils.JSONDict), data.(*jsonutils.JSONDict), request.UserCred)
	podName := params["<pod>"]
	container := params["<container>"]

	req := request.GetK8sClient().CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(request.GetNamespaceQuery().ToRequestParam()).
		SubResource("exec")

	vars := url.Values{}
	vars.Add("container", container)
	vars.Add("stdout", "1")
	vars.Add("stdin", "1")
	vars.Add("stderr", "1")
	vars.Add("tty", "1")
	vars.Add("command", "bash")
	//vars.Add("command", token)
	//vars.Add("command", context.ClusterName)

	//req.VersionedParams(&v1.PodExecOptions{
	//Container: container,
	//Command:   []string{"bash"},
	//Stdin:     true,
	//Stdout:    true,
	//Stderr:    true,
	//TTY:       true,
	//}, scheme.ParameterCodec)

	service, err := proxy.New(cluster)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	r.URL.Path = req.URL().Path
	r.URL.RawQuery = vars.Encode()
	service.ServeHTTP(w, r)*/
}
