package grafana

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/pkg/errors"
)

type Client interface {
	SetDebug(bool) Client
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

func (c *client) Post(ctx context.Context, url string, jsonData string) (jsonutils.JSONObject, error) {
	resp, err := httputils.Request(c.httpCli, ctx, httputils.POST, url, c.getHeader(), strings.NewReader(jsonData), c.debug)
	if err != nil {
		return nil, err
	}
	_, retData, err := httputils.ParseJSONResponse(jsonData, resp, err, c.debug)
	return retData, err
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
	_, err = c.Post(ctx, c.apiUrl+"/api/dashboards/import", string(body))
	if err != nil {
		return errors.Wrap(err, "import dashboard")
	}
	return nil
}
