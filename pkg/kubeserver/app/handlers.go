package app

import (
	"context"
	"fmt"
	"net/http"
	nethttppprof "net/http/pprof"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"

	"yunion.io/x/kubecomps/pkg/kubeserver/k8s"
	"yunion.io/x/kubecomps/pkg/kubeserver/k8s/common/model"
	k8sdispatcher "yunion.io/x/kubecomps/pkg/kubeserver/k8s/dispatcher"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	"yunion.io/x/kubecomps/pkg/kubeserver/usages"
)

func InitHandlers(app *appsrv.Application) {
	db.InitAllManagers()
	apiPrefix := "/api"
	taskman.AddTaskHandler(apiPrefix, app)
	usages.AddUsageHandler(apiPrefix, app)
	app.EnableProfiling()
	addPProfHandlers(apiPrefix, app)

	for _, man := range []db.IModelManager{
		taskman.TaskManager,
		taskman.SubTaskManager,
		taskman.TaskObjectManager,
		db.UserCacheManager,
		db.TenantCacheManager,
		db.SharedResourceManager,
		db.Metadata,
	} {
		db.RegisterModelManager(man)
	}

	for _, man := range []db.IModelManager{
		db.OpsLog,
		models.RepoManager,
		models.ClusterManager,
		models.ComponentManager,
		models.MachineManager,
		models.GetContainerRegistryManager(),

		// k8s cluster resource manager
		models.GetNodeManager(),
		models.GetNamespaceManager(),
		models.GetStorageClassManager(),
		models.GetClusterRoleManager(),
		models.GetClusterRoleBindingManager(),
		models.GetPVManager(),

		// k8s namespace resource manager
		models.GetPVCManager(),
		models.GetLimitRangeManager(),
		models.GetResourceQuotaManager(),
		models.GetRoleManager(),
		models.GetRoleBindingManager(),
		models.GetServiceManager(),
		models.GetIngressManager(),
		models.GetDeploymentManager(),
		models.GetStatefulSetManager(),
		models.GetDaemonSetManager(),
		models.GetReplicaSetManager(),
		models.GetJobManager(),
		models.GetCronJobManager(),
		models.GetPodManager(),
		models.ServiceAccountManager,
		models.GetSecretManager(),
		models.GetConfigMapManager(),
		models.GetReleaseManager(),

		// federated resources
		models.GetFedNamespaceManager(),
		models.GetFedClusterRoleManager(),
		models.GetFedClusterRoleBindingManager(),
		models.GetFedRoleManager(),
		models.GetFedRoleBindingManager(),
	} {
		db.RegisterModelManager(man)
		handler := db.NewModelHandler(man)
		dispatcher.AddModelDispatcher(apiPrefix, app, handler)
	}

	for _, man := range []db.IJointModelManager{
		models.ClusterComponentManager,

		// federated joint resources
		models.FedNamespaceClusterManager,
		models.FedClusterRoleClusterManager,
		models.FedClusterRoleBindingClusterManager,
		models.FedRoleClusterManager,
		models.FedRoleBindingClusterManager,
	} {
		db.RegisterModelManager(man)
		handler := db.NewJointModelHandler(man)
		dispatcher.AddJointModelDispatcher(apiPrefix, app, handler)
	}

	// k8s directly resource dispatcher
	v2Dispatcher := k8sdispatcher.NewK8sModelDispatcher(apiPrefix, app)
	for _, man := range []model.IK8sModelManager{
		models.GetEventManager(),

		// onecloud service operator resource manager
		models.GetVirtualMachineManager(),
		models.GetAnsiblePlaybookManager(),
		models.GetAnsiblePlaybookTemplateManager(),
	} {
		handler := model.NewK8SModelHandler(man)
		log.Infof("Dispatcher register k8s resource manager %q", man.KeywordPlural())
		v2Dispatcher.Add(handler)
	}

	k8s.AddHelmDispatcher(apiPrefix, app)
	k8s.AddRawResourceDispatcher(apiPrefix, app)
	k8s.AddMiscDispatcher(apiPrefix, app)
	addDefaultHandler(apiPrefix, app)
}

// addPProfHandlers exposes net/http/pprof under <apiPrefix>/debug/pprof/<name>.
// pprof.Index hard-codes "/debug/pprof/" when extracting the sub-profile name
// and only dispatches runtime/pprof.Lookup-based profiles (heap, goroutine,
// allocs, block, mutex, threadcreate). The four special handlers (cmdline,
// profile, symbol, trace) are exported as separate funcs and must be dispatched
// explicitly, so we do that here.
func addPProfHandlers(apiPrefix string, app *appsrv.Application) {
	stripPrefix := apiPrefix
	handler := func(_ context.Context, w http.ResponseWriter, r *http.Request) {
		r2 := r.Clone(r.Context())
		r2.URL.Path = strings.TrimPrefix(r2.URL.Path, stripPrefix)
		name := strings.TrimRight(strings.TrimPrefix(r2.URL.Path, "/debug/pprof/"), "/")
		switch name {
		case "cmdline":
			nethttppprof.Cmdline(w, r2)
		case "profile":
			nethttppprof.Profile(w, r2)
		case "symbol":
			nethttppprof.Symbol(w, r2)
		case "trace":
			nethttppprof.Trace(w, r2)
		default:
			// keep trailing slash off the path so Index doesn't pass "heap/" to Lookup
			r2.URL.Path = strings.TrimRight(r2.URL.Path, "/")
			nethttppprof.Index(w, r2)
		}
	}
	app.AddHandler("GET", fmt.Sprintf("%s/debug/pprof", apiPrefix),
		appsrv.WhitelistFilter(handler)).SetProcessNoTimeout()
	app.AddHandler("GET", fmt.Sprintf("%s/debug/pprof/<name>", apiPrefix),
		appsrv.WhitelistFilter(handler)).SetProcessNoTimeout()
	app.AddHandler("POST", fmt.Sprintf("%s/debug/pprof/<name>", apiPrefix),
		appsrv.WhitelistFilter(handler)).SetProcessNoTimeout()
}

func addDefaultHandler(apiPrefix string, app *appsrv.Application) {
	app.AddHandler("GET", fmt.Sprintf("%s/version", apiPrefix), appsrv.VersionHandler)
	app.AddHandler("GET", fmt.Sprintf("%s/stats", apiPrefix), appsrv.StatisticHandler)
	app.AddHandler("POST", fmt.Sprintf("%s/ping", apiPrefix), appsrv.PingHandler)
	app.AddHandler("GET", fmt.Sprintf("%s/ping", apiPrefix), appsrv.PingHandler)
	app.AddHandler("GET", fmt.Sprintf("%s/worker_stats", apiPrefix), appsrv.WorkerStatsHandler)
}
