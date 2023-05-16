package grafana

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
)

type Client interface {
	SetDebug(bool) Client
	ListFolders(ctx context.Context) ([]FolderHit, error)
	GetFolder(ctx context.Context, id int) (*Folder, error)
	CreateFolder(ctx context.Context, params CreateFolderParams) (*Folder, error)
	ImportDashboard(ctx context.Context, dashboard []byte, params ImportDashboardParams) error
}

type client struct {
	apiUrl   string
	user     string
	password string
	debug    bool

	httpCli *http.Client
}

func NewClient(apiUrl string, user string, password string) Client {
	c := &client{
		apiUrl:   apiUrl,
		user:     user,
		password: password,
		httpCli:  httputils.GetClient(true, 30*time.Second),
		debug:    false,
	}
	return c
}

func (c *client) SetDebug(debug bool) Client {
	c.debug = debug
	return c
}

func (c *client) basicAuth() string {
	auth := c.user + ":" + c.password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

func (c *client) getHeader() http.Header {
	h := http.Header{}
	h.Add("Authorization", c.basicAuth())
	h.Add("Accept", "application/json")
	h.Add("Content-Type", "application/json")
	h.Add("User-Agent", "kubeserver-grafana-cli")
	return h
}

func (c *client) Get(ctx context.Context, url string, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if query != nil {
		url = fmt.Sprintf("%s?%s", url, query.QueryString())
	}
	resp, err := httputils.Request(c.httpCli, ctx, httputils.GET, url, c.getHeader(), nil, c.debug)
	if err != nil {
		return nil, err
	}
	_, retData, err := httputils.ParseJSONResponse("", resp, err, c.debug)
	return retData, err
}

func (c *client) Post(ctx context.Context, url string, jsonData string) (jsonutils.JSONObject, error) {
	resp, err := httputils.Request(c.httpCli, ctx, httputils.POST, url, c.getHeader(), strings.NewReader(jsonData), c.debug)
	if err != nil {
		return nil, err
	}
	_, retData, err := httputils.ParseJSONResponse(jsonData, resp, err, c.debug)
	return retData, err
}

type CreateFolderParams struct {
	Title string `json:"title"`
}

type FolderHit struct {
	Id    int    `json:"id"`
	UId   string `json:"uid"`
	Title string `json:"title"`
}

type Folder struct {
	FolderHit

	CanAdmin  bool      `json:"canAdmin"`
	CanEdit   bool      `json:"canEdit"`
	CanSave   bool      `json:"canSave"`
	Created   time.Time `json:"created"`
	CreatedBy string    `json:"createdBy"`
	HasAcl    bool      `json:"hasAcl"`
	Updated   time.Time `json:"updated"`
	UpdatedBy string    `json:"updatedBy"`
	Url       string    `json:"url"`
	Version   int       `json:"version"`
}

type ImportDashboardInput struct {
	Type     string `json:"type"`
	PluginId string `json:"pluginId"`
	Name     string `json:"name"`
	Value    string `json:"value"`
}

type ImportDashboardParams struct {
	FolderId  int                    `json:"folderId"`
	Overwrite bool                   `json:"overwrite"`
	Inputs    []ImportDashboardInput `json:"inputs"`
}

type ImportDashboardData struct {
	ImportDashboardParams
	Dashboard map[string]interface{} `json:"dashboard"`
}

func (c *client) ApiUrl(suffix string) string {
	seg := "/"
	if strings.HasPrefix(suffix, "/") {
		seg = ""
	}
	return fmt.Sprintf("%s%s%s", c.apiUrl, seg, suffix)
}

func (c *client) ListFolders(ctx context.Context) ([]FolderHit, error) {
	resp, err := c.Get(ctx, c.ApiUrl("/api/folders"), nil)
	if err != nil {
		return nil, err
	}
	out := make([]FolderHit, 0)
	if err := resp.Unmarshal(&out); err != nil {
		return nil, errors.Wrapf(err, "unmarshal %q to folders", resp)
	}
	return out, nil
}

func (c *client) GetFolder(ctx context.Context, id int) (*Folder, error) {
	resp, err := c.Get(ctx, c.ApiUrl(fmt.Sprintf("/api/folders/id/%d", id)), nil)
	if err != nil {
		return nil, err
	}
	out := new(Folder)
	if err := resp.Unmarshal(&out); err != nil {
		return nil, errors.Wrapf(err, "unmarshal %q to folders", resp)
	}
	return out, nil
}

func (c *client) CreateFolder(ctx context.Context, params CreateFolderParams) (*Folder, error) {
	resp, err := c.Post(ctx, c.ApiUrl("/api/folders"), jsonutils.Marshal(params).String())
	if err != nil {
		return nil, err
	}
	out := new(Folder)
	if err := resp.Unmarshal(&out); err != nil {
		return nil, errors.Wrapf(err, "unmarshal %q to folders", resp)
	}
	return out, nil
}

func (c *client) ImportDashboard(ctx context.Context, dashboard []byte, params ImportDashboardParams) error {
	dashboardObj := make(map[string]interface{})
	if err := json.Unmarshal(dashboard, &dashboardObj); err != nil {
		return errors.Wrapf(err, "unmarshal dashboard")
	}
	data := &ImportDashboardData{
		ImportDashboardParams: params,
		Dashboard:             dashboardObj,
	}
	body, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "marshal data")
	}
	_, err = c.Post(ctx, c.ApiUrl("/api/dashboards/import"), string(body))
	if err != nil {
		return errors.Wrap(err, "import dashboard")
	}
	return nil
}
