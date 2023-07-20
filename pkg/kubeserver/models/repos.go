package models

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"helm.sh/helm/v3/pkg/repo"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/streamutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/helm"
	"yunion.io/x/kubecomps/pkg/kubeserver/options"
)

type SRepoManager struct {
	db.SStatusInfrasResourceBaseManager
}

var RepoManager *SRepoManager

func init() {
	RepoManager = &SRepoManager{SStatusInfrasResourceBaseManager: db.NewStatusInfrasResourceBaseManager(SRepo{}, "repos_tbl", "repo", "repos")}
	RepoManager.SetVirtualObject(RepoManager)
}

func (m *SRepoManager) InitializeData() error {
	// 填充 v2 没有 tenant_id 的 repo，默认变为 system project
	repos := []SRepo{}
	q := m.Query()
	q = q.Filter(sqlchemy.OR(
		sqlchemy.IsNullOrEmpty(q.Field("domain_id")),
		sqlchemy.IsNullOrEmpty(q.Field("domain_src")),
		sqlchemy.IsNullOrEmpty(q.Field("type")),
		sqlchemy.IsNullOrEmpty(q.Field("backend")),
	))
	if err := db.FetchModelObjects(m, q, &repos); err != nil {
		return errors.Wrap(err, "fetch empty project repos")
	}
	userCred := GetAdminCred()
	for _, r := range repos {
		tmpRepo := &r
		if _, err := db.Update(tmpRepo, func() error {
			if tmpRepo.DomainId == "" {
				tmpRepo.DomainId = userCred.GetProjectDomainId()
			}
			if tmpRepo.Type == "" {
				tmpRepo.Type = string(api.RepoTypeExternal)
			}
			if tmpRepo.Backend == "" {
				tmpRepo.Backend = string(api.RepoBackendCommon)
			}
			return nil
		}); err != nil {
			return errors.Wrapf(err, "update empty project repo %s", tmpRepo.GetName())
		}
	}

	return nil
}

type SRepo struct {
	db.SStatusInfrasResourceBase

	Url      string `width:"256" charset:"ascii" nullable:"false" create:"required" list:"user"`
	Username string `width:"256" charset:"ascii" nullable:"false" update:"user" list:"user"`
	Password string `width:"256" charset:"ascii" nullable:"false" update:"user"`
	Type     string `charset:"ascii" width:"128" create:"required" nullable:"true" list:"user"`
	Backend  string `charset:"ascii" width:"128" create:"required" nullable:"true" list:"user"`
}

func (man *SRepoManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (man *SRepoManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.RepoListInput) (*sqlchemy.SQuery, error) {
	q, err := man.SStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, err
	}
	if input.Type != "" {
		q = q.Equals("type", input.Type)
	}
	return q, nil
}

func (man *SRepoManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool) []api.RepoDetail {
	rows := make([]api.RepoDetail, len(objs))
	svRows := man.SStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range svRows {
		rObj := objs[i].(*SRepo)
		rlsCnt, err := rObj.GetReleaseCount()
		if err != nil {
			log.Errorf("Get repo release count error: %v", err)
		}
		detail := api.RepoDetail{
			StatusInfrasResourceBaseDetails: svRows[i],
			Url:                             rObj.Url,
			Type:                            rObj.Type,
			ReleaseCount:                    rlsCnt,
		}
		rows[i] = detail
	}
	return rows
}

func (man *SRepoManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeDomain
}

func (man *SRepoManager) GetRepoDataDir(projectId string) string {
	return path.Join(options.Options.HelmDataDir, projectId)
}

func (man *SRepoManager) GetClient(projectId string) (*helm.RepoClient, error) {
	dataDir := man.GetRepoDataDir(projectId)
	return helm.NewRepoClient(dataDir)
}

func (man *SRepoManager) GetChartClient(projectId string) *helm.ChartClient {
	dataDir := man.GetRepoDataDir(projectId)
	return helm.NewChartClient(dataDir)
}

func (man *SRepoManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.RepoCreateInput) (*api.RepoCreateInput, error) {
	shareInput, err := man.SStatusInfrasResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data.StatusInfrasResourceBaseCreateInput)
	if err != nil {
		return nil, err
	}
	data.StatusInfrasResourceBaseCreateInput = shareInput
	if data.Url == "" {
		return nil, httperrors.NewInputParameterError("Missing repo url")
	}
	if _, err := url.Parse(data.Url); err != nil {
		return nil, httperrors.NewNotAcceptableError("Invalid repo url: %v", err)
	}

	if data.Type == "" {
		data.Type = string(api.RepoTypeExternal)
	}
	if !utils.IsInStringArray(data.Type, []string{string(api.RepoTypeExternal), string(api.RepoTypeInternal)}) {
		return nil, httperrors.NewInputParameterError("Not support type %q", data.Type)
	}

	if data.Backend == "" {
		data.Backend = api.RepoBackendCommon
	}

	drv, err := man.GetDriver(data.Backend)
	if err != nil {
		return nil, errors.Wrapf(err, "get helm repo driver by backend %q", data.Backend)
	}
	data, err = drv.ValidateCreateData(ctx, userCred, ownerId, query, data)
	if err != nil {
		return nil, err
	}

	entry := man.ToEntry(data.Name, data.Url, data.Username, data.Password)
	cli, err := man.GetClient(ownerId.GetProjectDomainId())
	if err != nil {
		return nil, err
	}
	if err := cli.Add(entry); err != nil {
		log.Errorf("Add helm entry %#v error: %v", entry, err)
		if errors.Cause(err) == helm.ErrRepoAlreadyExists {
			return nil, httperrors.NewDuplicateResourceError("Backend helm repo name %s already exists, please specify a different name", data.Name)
		}
		return nil, httperrors.NewNotAcceptableError("Add helm repo %s failed", entry.URL)
	}

	return data, nil
}

func (man *SRepoManager) FetchRepoById(id string) (*SRepo, error) {
	repo, err := man.FetchById(id)
	if err != nil {
		return nil, err
	}
	return repo.(*SRepo), nil
}

func (man *SRepoManager) FetchRepoByIdOrName(userCred mcclient.IIdentityProvider, ident string) (*SRepo, error) {
	repo, err := man.FetchByIdOrName(userCred, ident)
	if err != nil {
		return nil, err
	}
	return repo.(*SRepo), nil
}

func (man *SRepoManager) ListRepos() ([]SRepo, error) {
	q := man.Query()
	repos := make([]SRepo, 0)
	err := db.FetchModelObjects(RepoManager, q, &repos)
	return repos, err
}

func (man *SRepoManager) ToEntry(name, url, username, password string) *repo.Entry {
	ret := &repo.Entry{
		Name:     name,
		URL:      url,
		Username: username,
		Password: password,
		// InsecureSkipTLSverify: true,
	}
	if strings.HasPrefix(url, "https") {
		ret.InsecureSkipTLSverify = true
	}
	return ret
}

func (r *SRepo) ToEntry() *repo.Entry {
	return RepoManager.ToEntry(r.Name, r.Url, r.Username, r.Password)
}

func (r *SRepo) GetReleaseCount() (int, error) {
	rlsCnt, err := GetReleaseManager().Query().Equals("repo_id", r.GetId()).CountWithError()
	if err != nil {
		return 0, errors.Wrapf(err, "get repo %s release count", r.GetName())
	}
	return rlsCnt, nil
}

func (r *SRepo) ValidateDeleteCondition(ctx context.Context, _ jsonutils.JSONObject) error {
	rlsCnt, err := r.GetReleaseCount()
	if err != nil {
		return errors.Wrap(err, "check release count")
	}
	if rlsCnt != 0 {
		return httperrors.NewNotAcceptableError("%d release use this repo", rlsCnt)
	}
	return nil
}

func (r *SRepo) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	cli, err := RepoManager.GetClient(r.DomainId)
	if err != nil {
		return err
	}
	if err := cli.Remove(r.Name); err != nil {
		return err
	}
	return r.SStatusInfrasResourceBase.Delete(ctx, userCred)
}

func (r *SRepo) PerformSync(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, r.DoSync()
}

func (r *SRepo) DoSync() error {
	cli, err := RepoManager.GetClient(r.DomainId)
	if err != nil {
		return err
	}
	entry := r.ToEntry()
	if err := cli.Add(entry); err != nil && errors.Cause(err) != helm.ErrRepoAlreadyExists {
		return err
	}
	return cli.Update(r.Name)
}

func (r *SRepo) GetType() api.RepoType {
	return api.RepoType(r.Type)
}

func (r *SRepo) GetChartClient() *helm.ChartClient {
	return RepoManager.GetChartClient(r.DomainId)
}

func (man *SRepoManager) GetDriver(backend api.RepoBackend) (IRepoDriver, error) {
	return GetRepoDriver(backend)
}

func (r *SRepo) GetBackend() api.RepoBackend {
	return api.RepoBackend(r.Backend)
}

func (r *SRepo) GetDriver() IRepoDriver {
	drv, err := RepoManager.GetDriver(r.GetBackend())
	if err != nil {
		panic(fmt.Sprintf("Get helm repo driver for %s/%s", r.GetId(), r.GetName()))
	}
	return drv
}

func (r *SRepo) PerformUploadChart(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	chartName, _ := query.GetString("file")
	if chartName == "" {
		chartName, _ = query.GetString("chart_name")
		if chartName == "" {
			return nil, httperrors.NewNotEmptyError("chart_name is empty")
		}
	}
	appParams := appsrv.AppContextGetParams(ctx)
	savedPath, err := saveImageFromStream(appParams.Request.Body, appParams.Request.ContentLength)
	defer func() {
		log.Infof("remove %s", savedPath)
		if savedPath != "" {
			os.RemoveAll(savedPath)
		}
	}()
	if err != nil {
		return nil, errors.Wrap(err, "save from stream")
	}
	drv := r.GetDriver()
	if _, err := drv.UploadChart(ctx, userCred, r, chartName, savedPath); err != nil {
		return nil, errors.Wrapf(err, "upload chart %s to %s", chartName, drv.GetBackend())
	}
	return nil, r.DoSync()
}

func (r *SRepo) GetDetailsDownloadChart(ctx context.Context, userCred mcclient.TokenCredential, input *api.RepoDownloadChartInput) (jsonutils.JSONObject, error) {
	if err := r.DoSync(); err != nil {
		return nil, errors.Wrap(err, "sync repo")
	}
	cli := r.GetChartClient()
	if input.ChartName == "" {
		return nil, httperrors.NewNotEmptyError("chart_name is empty")
	}
	chPath, err := cli.LocateChartPath(r.GetName(), input.ChartName, input.Version, r.ToEntry())
	if err != nil {
		return nil, errors.Wrapf(err, "LocateChartPath %s %s", input.ChartName, input.Version)
	}
	fStat, err := os.Stat(chPath)
	if err != nil {
		return nil, errors.Wrapf(err, "os.Stat %s", chPath)
	}
	f, err := os.Open(chPath)
	if err != nil {
		return nil, errors.Wrapf(err, "os.Open %s", chPath)
	}
	defer f.Close()
	fSize := fStat.Size()

	appParams := appsrv.AppContextGetParams(ctx)
	header := appParams.Response.Header()
	header.Set("Content-Length", strconv.FormatInt(fSize, 10))
	header.Set("Chart-Filename", filepath.Base(chPath))

	_, err = streamutils.StreamPipe(f, appParams.Response, false, nil)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	return nil, nil
}
