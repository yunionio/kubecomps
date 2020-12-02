package helm

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	//"helm.sh/helm/v3/pkg/downloader"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/options"
)

type Client struct {
	config      *action.Configuration
	setting     *cli.EnvSettings
	repoClient  *RepoClient
	chartClient *ChartClient
}

type IRepo interface {
	List() error
	Add() error
	Remove() error
	Update() error
}

type IRelease interface {
	Get() *action.Get
	List() *action.List
	Create(*api.ReleaseCreateInput) (*release.Release, error)
	Update(*api.ReleaseUpdateInput) (*release.Release, error)
	Install() *action.Install
	UnInstall() *action.Uninstall
	Upgrade() *action.Upgrade
	Rollback() *action.Rollback
	History() *action.History
	ReleaseContent(name string, version int) (*release.Release, error)
}

type IChart interface {
	List() *action.ChartList
	Show() error
}

func NewClient(kubeConfig string, namespace string, debug bool) (*Client, error) {
	config := new(action.Configuration)
	setting := NewRepoConfig(options.Options.HelmDataDir).GetSetting()
	setting.KubeConfig = kubeConfig
	debugF := func(format string, v ...interface{}) {
		if debug {
			format = fmt.Sprintf("[helm-debug] %s", format)
			log.Infof(format, v...)
		}
	}

	// revise namespace
	cliGetter := setting.RESTClientGetter().(*genericclioptions.ConfigFlags)
	cliGetter.Namespace = &namespace

	if err := config.Init(cliGetter, namespace, "", debugF); err != nil {
		return nil, errors.Wrap(err, "init configuration")
	}
	repoClient, err := NewRepoClient(options.Options.HelmDataDir)
	if err != nil {
		return nil, errors.Wrap(err, "create repo client")
	}
	return &Client{
		config:      config,
		setting:     setting,
		repoClient:  repoClient,
		chartClient: NewChartClient(options.Options.HelmDataDir),
	}, nil
}

func (c *Client) GetConfig() *action.Configuration {
	return c.config
}

func (c *Client) GetSetting() *cli.EnvSettings {
	return c.setting
}

func (c *Client) Release() IRelease {
	return &releaseClient{
		client:      c,
		repoClient:  c.repoClient,
		chartClient: c.chartClient,
	}
}

func (c *Client) Chart() IChart {
	return nil
}

type releaseClient struct {
	client      *Client
	repoClient  *RepoClient
	chartClient *ChartClient
}

func (r releaseClient) Get() *action.Get {
	return action.NewGet(r.client.GetConfig())
}

func (r releaseClient) List() *action.List {
	return action.NewList(r.client.GetConfig())
}

func (r releaseClient) Install() *action.Install {
	return action.NewInstall(r.client.GetConfig())
}

func (r releaseClient) UnInstall() *action.Uninstall {
	return action.NewUninstall(r.client.GetConfig())
}

func (r releaseClient) Upgrade() *action.Upgrade {
	return action.NewUpgrade(r.client.GetConfig())
}

func (r releaseClient) Rollback() *action.Rollback {
	return action.NewRollback(r.client.GetConfig())
}

func (r releaseClient) History() *action.History {
	return action.NewHistory(r.client.GetConfig())
}

var (
	// errMissingChart indicates that a chart was not provided.
	errMissingChart = errors.Error("no chart provided")
	// errMissingRelease indicates that a release (name) was not provided.
	errMissingRelease = errors.Error("no release provided")
	// errInvalidRevision indicates that an invalid release revision number was provided
	errInvalidRevision = errors.Error("invalid release revision")
	// errInvalidName indicates that an invalid release name was provided
	errInvalidName = errors.Error("invalid release name, must match regex ^(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])+$ and the length must not longer than 53")
)

// ValidName is a regular expression for names.
//
// According to the Kubernetes help text, the regular expression it uses is:
//
//  (([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?
//
// We modified that. First, we added start and end delimiters. Second, we changed
// the final ? to + to require that the pattern match at least once. This modification
// prevents an empty string from matching.
var ValidName = regexp.MustCompile("^(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])+$")

const releaseNameMaxLen = 53

func ValidateReleaseName(releaseName string) error {
	if releaseName == "" {
		return errMissingRelease
	}
	if !ValidName.MatchString(releaseName) || (len(releaseName) > releaseNameMaxLen) {
		return errInvalidName
	}
	return nil
}

func (r releaseClient) ReleaseContent(name string, version int) (*release.Release, error) {
	if err := ValidateReleaseName(name); err != nil {
		return nil, errors.Wrapf(err, "release name is invalid %q", name)
	}

	if version <= 0 {
		return r.client.config.Releases.Last(name)
	}
	return r.client.config.Releases.Get(name, version)
}

func (r releaseClient) getValueOptions(valueStr string, sets map[string]string) (map[string]interface{}, error) {
	p := getter.All(r.client.GetSetting())
	tmpValueF, err := ioutil.TempFile("", "helm-release-value")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpValueF.Name())

	if _, err := tmpValueF.Write([]byte(valueStr)); err != nil {
		return nil, err
	}
	if err := tmpValueF.Close(); err != nil {
		return nil, err
	}
	setStrs := make([]string, 0)
	for k, v := range sets {
		setStrs = append(setStrs, fmt.Sprintf("%s=%s", k, v))
	}
	ret := values.Options{
		ValueFiles: []string{tmpValueF.Name()},
		Values:     setStrs,
	}
	return ret.MergeValues(p)
}

func (r releaseClient) Create(input *api.ReleaseCreateInput) (*release.Release, error) {
	if input.Version == "" {
		input.Version = ">0.0.0-0"
	}

	cp, err := r.chartClient.Show(input.Repo, input.ChartName, input.Version)
	if err != nil {
		return nil, err
	}
	vals, err := r.getValueOptions(input.Values, input.Sets)
	if err != nil {
		return nil, err
	}
	validInstallableChart, err := isChartInstallable(cp)
	if !validInstallableChart {
		return nil, err
	}

	if req := cp.Metadata.Dependencies; req != nil {
		// If CheckDependencies returns an error, we have unfulfilled dependencies.
		// As of Helm 2.4.0, this is treated as a stopping condition:
		//  https://github.com/helm/helm/issues/2209
		if err := action.CheckDependencies(cp, req); err != nil {
			//man := &downloader.Manager{
			//}
			return nil, err
		}
	}

	client := r.Install()
	client.Version = input.Version
	client.Namespace = input.Namespace
	client.ReleaseName = input.ReleaseName

	return client.Run(cp, vals)
}

// isChartInstallable validates if a chart can be installed
//
// Application chart type is only installable
func isChartInstallable(ch *chart.Chart) (bool, error) {
	switch ch.Metadata.Type {
	case "", "application":
		return true, nil
	}
	return false, errors.Errorf("%s charts are not installable", ch.Metadata.Type)
}

func (r releaseClient) Update(input *api.ReleaseUpdateInput) (*release.Release, error) {
	if input.Version == "" {
		input.Version = ">0.0.0-0"
	}

	vals, err := r.getValueOptions(input.Values, input.Sets)
	if err != nil {
		return nil, err
	}
	cp, err := r.chartClient.Show(input.Repo, input.ChartName, input.Version)
	if err != nil {
		return nil, err
	}

	// Check chart dependencies to make sure all are present in /charts
	if req := cp.Metadata.Dependencies; req != nil {
		if err := action.CheckDependencies(cp, req); err != nil {
			return nil, err
		}
	}

	cli := r.Upgrade()
	cli.Version = input.Version
	cli.Namespace = input.Namespace
	return cli.Run(input.ReleaseName, cp, vals)
}
