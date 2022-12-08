package app

import (
	"context"
	"net"
	"os"
	"strconv"

	_ "github.com/go-sql-driver/mysql"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	app_commmon "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/pkg/util/runtime"
	"yunion.io/x/pkg/util/signalutils"

	"yunion.io/x/kubecomps/pkg/kubeserver/constants"
	"yunion.io/x/kubecomps/pkg/kubeserver/controllers"
	"yunion.io/x/kubecomps/pkg/kubeserver/helm"
	"yunion.io/x/kubecomps/pkg/kubeserver/initial"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	"yunion.io/x/kubecomps/pkg/kubeserver/options"
	_ "yunion.io/x/kubecomps/pkg/kubeserver/policy"
	"yunion.io/x/kubecomps/pkg/kubeserver/server"
)

func prepareEnv() {
	os.Unsetenv("SSH_AUTH_SOCK")
	os.Unsetenv("SSH_AGENT_PID")
	os.Setenv("DISABLE_HTTP2", "true")

	// for ansible
	os.Setenv("PATH", "$PATH:/usr/bin/")
	os.Setenv("ANSIBLE_HOST_KEY_CHECKING", "False")

	common_options.ParseOptions(&options.Options, os.Args, "kubeserver.conf", constants.ServiceType)
	runtime.ReallyCrash = false
	helm.InitEnv(options.Options.HelmDataDir)
}

func Run(ctx context.Context) error {
	opt := &options.Options
	prepareEnv()
	cloudcommon.InitDB(&opt.DBOptions)
	defer cloudcommon.CloseDB()

	app := app_commmon.InitApp(&opt.BaseOptions, true)
	InitHandlers(app)

	app_commmon.InitAuth(&opt.CommonOptions, func() {})
	common_options.StartOptionManager(opt, opt.ConfigSyncPeriodSeconds, constants.ServiceType, constants.ServiceVersion, options.OnOptionsChange)

	if db.CheckSync(options.Options.AutoSyncTable) {
		for _, initDBFunc := range []func() error{
			models.InitDB,
		} {
			err := initDBFunc()
			if err != nil {
				log.Fatalf("Init models error: %v", err)
			}
		}
	} else {
		log.Fatalf("Fail sync db")
	}

	go func() {
		log.Infof("Auth complete, start controllers.")
		controllers.Start()
	}()

	httpsAddr := net.JoinHostPort(opt.Address, strconv.Itoa(opt.HttpsPort))

	if err := models.GetClusterManager().SyncClustersFromCloud(ctx); err != nil {
		// log.Fatalf("Sync clusters from cloud: %v", err)
		log.Errorf("Sync clusters from cloud: %v", err)
	}

	cron := cronman.InitCronJobManager(true, options.Options.CronJobWorkerCount)
	initial.InitClient(cron)
	cron.Start()
	defer cron.Stop()

	if err := models.GetClusterManager().RegisterSystemCluster(); err != nil {
		log.Fatalf("Register system cluster %v", err)
	}

	if err := server.Start(httpsAddr, app); err != nil {
		return err
	}
	return nil
}

func init() {
	signalutils.SetDumpStackSignal()
	signalutils.StartTrap()
}
