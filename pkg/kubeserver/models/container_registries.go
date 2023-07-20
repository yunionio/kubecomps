package models

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/streamutils"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/container_registries/client"
	"yunion.io/x/kubecomps/pkg/utils/registry"
)

type SContainerRegistryManager struct {
	db.SStatusInfrasResourceBaseManager
}

var (
	containerRegistryManager *SContainerRegistryManager
	imgStreamingWorkerMan    = appsrv.NewWorkerManager("image_streaming_worker", 10, 1024, true)
)

func GetContainerRegistryManager() *SContainerRegistryManager {
	if containerRegistryManager == nil {
		containerRegistryManager = &SContainerRegistryManager{
			SStatusInfrasResourceBaseManager: db.NewStatusInfrasResourceBaseManager(SContainerRegistry{}, "container_registries_tbl", "container_registry", "container_registries"),
		}
		containerRegistryManager.SetVirtualObject(containerRegistryManager)
	}
	return containerRegistryManager
}

func init() {
	GetContainerRegistryManager()
}

type SContainerRegistry struct {
	db.SStatusInfrasResourceBase

	Url    string               `width:"256" charset:"ascii" nullable:"false" create:"required" update:"user" list:"user"`
	Type   string               `charset:"ascii" width:"128" create:"required" nullable:"true" list:"user"`
	Config jsonutils.JSONObject `nullable:"true" create:"optional"`
}

func (man *SContainerRegistryManager) AddDispatcher(prefix string, app *appsrv.Application, manager dispatcher.IModelDispatchHandler) {
	prefix = fmt.Sprintf("%s/%s/<resid>/", prefix, man.KeywordPlural())
	// app.AddHandler2("GET", fmt.Sprintf("%s/images", prefix),
	//	manager.Filter(man.getImages), nil, "list_images", nil)
}

func (man *SContainerRegistryManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.ContainerRegistryListInput) (*sqlchemy.SQuery, error) {
	q, err := man.SStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, err
	}
	if input.Type != "" {
		q = q.Equals("type", input.Type)
	}
	return q, nil
}

func (man *SContainerRegistryManager) GetDriver(rType api.ContainerRegistryType) (IContainerRegistryDriver, error) {
	return GetContainerRegistryDriver(rType)
}

func (man *SContainerRegistryManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.ContainerRegistryCreateInput) (*api.ContainerRegistryCreateInput, error) {
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

	driver, err := man.GetDriver(data.Type)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Get driver by type: %q", data.Type)
	}

	data, err = driver.ValidateCreateData(ctx, userCred, ownerId, query, data)
	if err != nil {
		return nil, errors.Wrapf(err, "validate %q create data", driver.GetType())
	}

	rgCli, err := driver.GetDockerRegistryClient(data.Url, &data.Config)
	if err != nil {
		return nil, errors.Wrapf(err, "get docker registry client on %q", driver.GetType())
	}

	if err := rgCli.Ping(ctx); err != nil {
		return nil, errors.Wrapf(err, "ping docker registry on %q", driver.GetType())
	}

	return data, err
}

func (r *SContainerRegistry) GetConfig() (*api.ContainerRegistryConfig, error) {
	conf := new(api.ContainerRegistryConfig)
	if err := r.Config.Unmarshal(conf); err != nil {
		return nil, err
	}
	return conf, nil
}

func (r *SContainerRegistry) GetType() api.ContainerRegistryType {
	return api.ContainerRegistryType(r.Type)
}

func (r *SContainerRegistry) GetDriver() IContainerRegistryDriver {
	drv, err := GetContainerRegistryManager().GetDriver(r.GetType())
	if err != nil {
		panic(fmt.Sprintf("Get container registry driver for %s/%s", r.GetId(), r.GetName()))
	}
	return drv
}

func (r *SContainerRegistry) GetDockerRegistryClient() (client.Client, error) {
	conf, err := r.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "get config")
	}
	return r.GetDriver().GetDockerRegistryClient(r.Url, conf)
}

func mergeQueryParams(params map[string]string, query jsonutils.JSONObject, excludes ...string) jsonutils.JSONObject {
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
func (m *SContainerRegistryManager) getImages(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	//params, query, body := appsrv.FetchEnv(ctx, w, r)
	//result, err := m.(ctx, params["<resid>"], mergeQueryParams(params, query, "<resid>"), false)
	log.Errorln("==============getImages called")
}

func (r *SContainerRegistry) GetDetailsImages(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	rgCli, err := r.GetDockerRegistryClient()
	if err != nil {
		return nil, errors.Wrap(err, "GetDockerRegistryClient")
	}
	return rgCli.ListImages(ctx)
}

func (r *SContainerRegistry) GetDetailsImageTags(ctx context.Context, userCred mcclient.TokenCredential, query *api.ContainerRegistryGetImageTagsInput) (jsonutils.JSONObject, error) {
	if query.Repository == "" {
		return nil, httperrors.NewNotEmptyError("repository is empty")
	}
	rgCli, err := r.GetDockerRegistryClient()
	if err != nil {
		return nil, errors.Wrap(err, "GetDockerRegistryClient")
	}
	return rgCli.ListImageTags(ctx, query.Repository)
}

func (m *SContainerRegistryManager) CustomizeHandlerInfo(info *appsrv.SHandlerInfo) {
	m.SStatusInfrasResourceBaseManager.CustomizeHandlerInfo(info)

	switch info.GetName(nil) {
	case "perform_action", "get_specific":
		info.SetProcessTimeout(time.Minute * 120).SetWorkerManager(imgStreamingWorkerMan)
	}
}

func (r *SContainerRegistry) PerformUploadImage(ctx context.Context, userCred mcclient.TokenCredential, query, data api.ContainerRegistryUploadImageInput) (*client.ImageMetadata, error) {
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
	return r.uploadImage(ctx, savedPath, data)
}

func saveImageFromStream(reader io.Reader, totalSize int64) (string, error) {
	imgName := stringutils.UUID4()
	tarPath := fmt.Sprintf("/tmp/%s", imgName)
	fp, err := os.Create(tarPath)
	if err != nil {
		return "", err
	}
	defer fp.Close()
	lastSaveTime := time.Now()
	sp, err := streamutils.StreamPipe(reader, fp, false, func(saved int64) {
		now := time.Now()
		if now.Sub(lastSaveTime) > 5*time.Second {
			log.Infof("saved %d", totalSize)
			lastSaveTime = now
		}
	})
	log.Infof("---sp checksum: %v, error: %v", sp.CheckSum, err)
	return tarPath, err
}

func (r *SContainerRegistry) uploadImage(ctx context.Context, imgPath string, input api.ContainerRegistryUploadImageInput) (*client.ImageMetadata, error) {
	dstPath := fmt.Sprintf("%s.tar", imgPath)
	isGzip, err := registry.IsGzipFile(imgPath)
	if err != nil {
		return nil, errors.Wrapf(err, "check %q is gzip file", imgPath)
	}
	if isGzip {
		log.Infof("%q is gzip, decompress it", imgPath)
		if err := registry.UnZip(imgPath, dstPath); err != nil {
			return nil, errors.Wrapf(err, "unzip %q", imgPath)
		}
	} else {
		if err := os.Rename(imgPath, dstPath); err != nil {
			return nil, errors.Wrapf(err, "rename %q to %q", imgPath, dstPath)
		}
	}
	imgPath = dstPath
	defer func() {
		if err := os.RemoveAll(imgPath); err != nil {
			log.Infof("try to remove %q", imgPath)
			if err != nil {
				log.Errorf("remove %q: %v", imgPath, err)
			}
			log.Infof("%q removed", imgPath)
		}
	}()

	cli, err := r.GetDockerRegistryClient()
	if err != nil {
		return nil, errors.Wrap(err, "get docker registry client")
	}
	meta, err := cli.AnalysisImageTarMetadata(imgPath)
	if err != nil {
		return nil, errors.Wrapf(err, "analysis image metadata")
	}
	if input.Tag != "" {
		meta.Ref.Tag = input.Tag
	}
	if input.Repository != "" {
		meta.Ref.Repository = input.Repository
	}

	driver := r.GetDriver()
	conf, _ := r.GetConfig()
	if err := driver.PreparePushImage(ctx, r.Url, conf, meta); err != nil {
		return nil, errors.Wrapf(err, "prepare push image to %q", driver.GetType())
	}

	if err := cli.PushImage(ctx, meta, imgPath); err != nil {
		return nil, errors.Wrapf(err, "push image by input: %s", jsonutils.Marshal(input))
	}
	return meta, nil
}

func (r *SContainerRegistry) GetDetailsDownloadImage(ctx context.Context, userCred mcclient.TokenCredential, query api.ContainerRegistryDownloadImageInput) (jsonutils.JSONObject, error) {
	if query.ImageName == "" {
		return nil, httperrors.NewNotEmptyError("image name required")
	}
	if query.Tag == "" {
		return nil, httperrors.NewNotEmptyError("image tag required")
	}
	drv := r.GetDriver()
	conf, _ := r.GetConfig()
	savedPath, err := drv.DownloadImage(ctx, r.Url, conf, query)
	if err != nil {
		return nil, errors.Wrap(err, "download image")
	}

	fStat, err := os.Stat(savedPath)
	if err != nil {
		return nil, errors.Wrapf(err, "os.Stat %s", savedPath)
	}
	f, err := os.Open(savedPath)
	if err != nil {
		return nil, errors.Wrapf(err, "os.Open %s", savedPath)
	}
	defer f.Close()
	fSize := fStat.Size()

	appParams := appsrv.AppContextGetParams(ctx)
	header := appParams.Response.Header()
	header.Set("Content-Length", strconv.FormatInt(fSize, 10))
	header.Set("Image-Filename", filepath.Base(savedPath))

	defer func() {
		for _, sp := range []string{
			savedPath,
			strings.TrimSuffix(savedPath, ".gz"),
		} {
			if err := os.RemoveAll(sp); err != nil {
				log.Infof("try to remove %q", sp)
				if err != nil {
					log.Errorf("remove %q: %v", sp, err)
				}
				log.Infof("%q removed", sp)
			}
		}
	}()

	_, err = streamutils.StreamPipe(f, appParams.Response, false, nil)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	return nil, nil
}
