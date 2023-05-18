package request

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/util/httputils"
)

func Get(endpoint string, token string, url string, header http.Header, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return JSONRequest(endpoint, token, "GET", url, header, body)
}

func Post(endpoint string, token string, url string, header http.Header, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return JSONRequest(endpoint, token, "POST", url, header, body)
}

func Put(endpoint string, token string, url string, header http.Header, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return JSONRequest(endpoint, token, "PUT", url, header, body)
}

func Delete(endpoint string, token string, url string, header http.Header, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return JSONRequest(endpoint, token, "DELETE", url, header, body)
}

func JSONRequest(endpoint string, token string, method string, url string, header http.Header, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ctx := context.Background()
	cli := httputils.GetDefaultClient()
	_, ret, err := httputils.JSONRequest(cli, ctx, httputils.THttpMethod(method), JoinUrl(endpoint, url), GetDefaultHeader(header, token), body, true)
	return ret, err
}

func JoinUrl(baseUrl, path string) string {
	base, version := mcclient.SplitVersionedURL(baseUrl)
	if len(version) > 0 {
		if strings.HasPrefix(path, fmt.Sprintf("/%s/", version)) {
			baseUrl = base
		}
	}
	return fmt.Sprintf("%s%s", baseUrl, path)
}

func GetDefaultHeader(header http.Header, token string) http.Header {
	if len(token) > 0 {
		if header == nil {
			header = http.Header{}
		}
		header.Add("X-Auth-Token", token)
	}
	return header
}
