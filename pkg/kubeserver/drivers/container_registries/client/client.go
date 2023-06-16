package client

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/regclient/regclient"
	"github.com/regclient/regclient/config"
	"github.com/regclient/regclient/types/ref"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/kubecomps/pkg/kubeserver/request"
	"yunion.io/x/kubecomps/pkg/utils/registry"
)

type Client interface {
	Ping(ctx context.Context) error

	ListImages(ctx context.Context) (jsonutils.JSONObject, error)

	ListImageTags(ctx context.Context, image string) (jsonutils.JSONObject, error)

	AnalysisImageTarMetadata(tarPath string) (*ImageMetadata, error)

	PushImage(ctx context.Context, input *ImageMetadata, tarPath string) error
}

// DockerAuthConfig contains authorization information for connecting to a registry.
// the value of Username and Password can be empty for accessing the registry anonymously
type DockerAuthConfig struct {
	Username string
	Password string
	// IdentityToken can be used as a refresh_token in place of username and
	// password to obtain the bearer/access token in oauth2 flow. If identity
	// token is set, password should not be set.
	// Ref: https://docs.docker.com/registry/spec/auth/oauth/
	IdentityToken string
}

type client struct {
	regUrl     string
	regHost    string
	authConfig DockerAuthConfig
	rc         *regclient.RegClient
}

func NewClient(regUrl string, authConf DockerAuthConfig) (Client, error) {
	regURL, err := url.Parse(regUrl)
	if err != nil {
		return nil, errors.Wrapf(err, "parse registry url %s", regUrl)
	}
	regHost := regURL.Host

	rc := regclient.New(regclient.WithConfigHost(config.Host{
		Name:      regHost,
		Hostname:  regHost,
		TLS:       config.TLSInsecure,
		ReqPerSec: 100,
		User:      authConf.Username,
		Pass:      authConf.Password,
		RepoAuth:  true,
	}))
	return &client{
		regUrl:     regUrl,
		regHost:    regHost,
		authConfig: authConf,
		rc:         rc,
	}, nil
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func (c client) getHeader(header http.Header) http.Header {
	if header == nil {
		header = http.Header{}
	}
	if c.authConfig.Username != "" {
		header.Set("Authorization", fmt.Sprintf("Basic "+basicAuth(c.authConfig.Username, c.authConfig.Password)))
	}
	return header
}

func (c client) Request(ctx context.Context, method httputils.THttpMethod, urlPath string, header http.Header, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	cli := httputils.GetDefaultClient()
	_, ret, err := httputils.JSONRequest(cli, ctx, method, request.JoinUrl(c.regUrl, urlPath), c.getHeader(header), body, true)
	return ret, err
}

func (c client) Get(ctx context.Context, urlPath string, header http.Header, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return c.Request(ctx, httputils.GET, urlPath, header, body)
}

func (c client) Ping(ctx context.Context) error {
	//TODO implement me
	pingUrl := "/v2/"
	_, err := c.Get(ctx, pingUrl, nil, nil)
	return err
}

func (c client) ListImages(ctx context.Context) (jsonutils.JSONObject, error) {
	catalogUrl := "/v2/_catalog"
	resp, err := c.Get(ctx, catalogUrl, nil, nil)
	log.Errorf("resp images: %s", resp.PrettyString())
	return resp, err
}

func (c client) ListImageTags(ctx context.Context, image string) (jsonutils.JSONObject, error) {
	tagsUrl := "/v2/%s/tags/list"
	image = url.QueryEscape(image)
	resp, err := c.Get(ctx, fmt.Sprintf(tagsUrl, image), nil, nil)
	if err != nil {
		return nil, err
	}
	log.Errorf("resp tags: %s", resp.PrettyString())
	return resp, nil
}

func (c client) AnalysisTar(tarPath string) (*registry.TarReadData, error) {
	return registry.AnalysisTar(tarPath)
}

type ImageMetadata struct {
	Ref ref.Ref
}

func newImageMetadata(data *registry.TarReadData) (*ImageMetadata, error) {
	manifests := data.GetDockerManifestList()
	if len(manifests) == 0 {
		return nil, errors.Errorf("not found manifests")
	}
	if len(manifests[0].RepoTags) == 0 {
		return nil, errors.Errorf("not found RepoTags from")
	}
	repoTag := manifests[0].RepoTags[0]
	ref, err := ref.New(repoTag)
	if err != nil {
		return nil, errors.Errorf("ref.New(%s)", repoTag)
	}
	return &ImageMetadata{Ref: ref}, nil
}

func (c client) AnalysisImageTarMetadata(tarPath string) (*ImageMetadata, error) {
	data, err := c.AnalysisTar(tarPath)
	if err != nil {
		return nil, errors.Wrapf(err, "analysis tar %q data", tarPath)
	}
	meta, err := newImageMetadata(data)
	if err != nil {
		return nil, errors.Wrapf(err, "new image metadata")
	}
	return meta, nil
}

func (c client) PushImage(ctx context.Context, meta *ImageMetadata, tarPath string) error {
	ref := meta.Ref
	ref.Registry = c.regHost

	f, err := os.Open(tarPath)
	if err != nil {
		return errors.Wrapf(err, "open file %q", tarPath)
	}
	defer f.Close()
	log.Infof("import %q to %#v", tarPath, ref)
	return c.rc.ImageImport(ctx, ref, f)
}
