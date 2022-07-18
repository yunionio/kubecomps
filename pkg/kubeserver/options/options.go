package options

import (
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
)

var (
	Options KubeServerOptions
)

type KubeServerOptions struct {
	common_options.DBOptions
	common_options.CommonOptions

	TlsCertFile       string `help:"File containing the default x509 cert file"`
	TlsPrivateKeyFile string `help:"Tls private key"`
	HttpsPort         int    `help:"The https port that the service runs on" default:"8443"`

	HelmDataDir         string `help:"Helm data directory" default:"/opt/cloud/workspace/helm"`
	RepoRefreshDuration int    `help:"Helm repo auto refresh duration, default 5 mins" default:"5"`

	EnableDefaultLimitRange bool `help:"Enable default namespace limit range" default:"false"`

	// GuestDefaultTemplate string `help:"Guest kubernetes default image id" default:"k8s-centos7-base.qcow2"`
	// GuestDefaultTemplate string `help:"Guest kubernetes default image id" default:"CentOS-7.6.1810-20190430.qcow2"`

	// hack: repo charts ignore regexp
	ChartIgnores []string `help:"Repo chart ignore regexp config, e.g. '^vm-repo:onecloud-.*:0.2.0$'"`

	//k8s
	DownloadFileURL string `help:"k8s depends on the binary file download url" default:"https://iso.yunion.cn"`
	//iamge repo
	ImageRepo string `help:"k8s depends on the image repo" default:"registry.cn-beijing.aliyuncs.com/yunionio"`
	//docker info
	DockerUser     string `help:"docker login user name"`
	DockerPassword string `help:"docker login password"`
	DockerHost     string `help:"docker login host"`
}

func OnOptionsChange(oldO, newO interface{}) bool {
	oldOpts := oldO.(*KubeServerOptions)
	newOpts := newO.(*KubeServerOptions)
	changed := false
	if common_options.OnCommonOptionsChange(&oldOpts.CommonOptions, &newOpts.CommonOptions) {
		changed = true
	}
	return changed
}
