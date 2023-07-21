package container_registries

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
	"yunion.io/x/kubecomps/pkg/kubeserver/drivers/container_registries/client"
	"yunion.io/x/kubecomps/pkg/kubeserver/models"
	"yunion.io/x/kubecomps/pkg/utils/skopeo"
)

func init() {
	models.RegisterContainerRegistryDriver(newCommonImpl())
}

func newCommonImpl() models.IContainerRegistryDriver {
	return new(commonImpl)
}

type commonImpl struct{}

func (c commonImpl) GetType() api.ContainerRegistryType {
	return api.ContainerRegistryTypeCommon
}

func (c commonImpl) GetDockerRegistryClient(url string, config *api.ContainerRegistryConfig) (client.Client, error) {
	if config.Common == nil {
		return nil, errors.Errorf("harbor config is nil")
	}
	return client.NewClient(url, client.DockerAuthConfig{
		Username: config.Common.Username,
		Password: config.Common.Password,
	})
}

func (c commonImpl) GetPathPrefix(inputUrl string) (string, error) {
	regUrl, err := url.Parse(inputUrl)
	if err != nil {
		return "", httperrors.NewInputParameterError("Invalid url: %q", inputUrl)
	}
	if regUrl.Path == "" {
		return "", httperrors.NewInputParameterError("Url %q must with path prefix used for a repository namespace", inputUrl)
	}
	return regUrl.Path, nil
}

func (c commonImpl) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.ContainerRegistryCreateInput) (*api.ContainerRegistryCreateInput, error) {
	config := data.Config.Common
	if config == nil {
		return nil, httperrors.NewInputParameterError("Configuration of common is nil")
	}
	if _, err := c.GetPathPrefix(data.Url); err != nil {
		return nil, err
	}
	cli, err := c.GetDockerRegistryClient(data.Url, &data.Config)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Get docker registry client: %v", err)
	}
	if err := cli.Ping(ctx); err != nil {
		return nil, httperrors.NewInputParameterError("Ping docker registry %q: %v", data.Url, err)
	}
	return data, nil
}

func (c commonImpl) PreparePushImage(ctx context.Context, url string, conf *api.ContainerRegistryConfig, meta *client.ImageMetadata) error {
	parts := strings.Split(meta.Ref.Repository, "/")
	meta.Ref.Repository = parts[len(parts)-1]
	return nil
}

func (c commonImpl) DownloadImage(ctx context.Context, regUrl string, conf *api.ContainerRegistryConfig, input api.ContainerRegistryDownloadImageInput) (string, error) {
	srcUrl := strings.TrimPrefix(strings.TrimPrefix(regUrl, "http://"), "https://")
	imgName := fmt.Sprintf("%s:%s", input.ImageName, input.Tag)
	savedPath := fmt.Sprintf("/tmp/%s.tar", strings.ReplaceAll(imgName, ":", "-"))
	imageUrl := filepath.Join(srcUrl, imgName)
	copyParams := &skopeo.CopyParams{
		SrcTLSVerify: false,
		SrcUsername:  conf.Common.Username,
		SrcPassword:  conf.Common.Password,
		SrcPath:      imageUrl,
		TargetPath:   savedPath,
	}
	if err := skopeo.NewSkopeo().Copy(copyParams); err != nil {
		return "", errors.Wrapf(err, "copy container image to local path %s", savedPath)
	}
	gzFilePath := fmt.Sprintf("%s.gz", savedPath)
	f, err := os.Create(gzFilePath)
	if err != nil {
		return "", errors.Wrapf(err, "create %s", gzFilePath)
	}
	defer f.Close()
	w, err := gzip.NewWriterLevel(f, gzip.BestSpeed)
	if err != nil {
		return "", errors.Wrap(err, "gzip.NewWriterLevel")
	}
	defer w.Close()
	savedFile, err := os.Open(savedPath)
	if err != nil {
		return "", errors.Wrapf(err, "open %s", savedPath)
	}
	defer savedFile.Close()
	if _, err := io.Copy(w, savedFile); err != nil {
		return "", errors.Wrapf(err, "gzip %s to %s", savedPath, gzFilePath)
	}
	// Flush the gzip writer to ensure all data is written
	w.Flush()

	return gzFilePath, nil
}
