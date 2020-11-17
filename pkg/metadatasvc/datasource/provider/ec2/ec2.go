package ec2

import (
	"context"
	"io/ioutil"
	"net"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/util/httputils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/metadatasvc/datasource"
	"yunion.io/x/kubecomps/pkg/metadatasvc/datasource/provider"
	"yunion.io/x/kubecomps/pkg/metadatasvc/metadata"
)

const (
	LatestSupportedVersion                    = "2016-09-02"
	MetadataURL            provider.FormatURL = "http://169.254.169.254/%s/%s/%s"
)

func init() {
	datasource.RegisterProvider(&metadataService{
		URL: MetadataURL,
	})
}

type metadataService struct {
	URL provider.FormatURL
}

func (s *metadataService) fetchMetadataTextAttr(ctx context.Context, attr string) (string, error) {
	version := LatestSupportedVersion

	url := s.URL.Fill(version, "meta-data", attr)
	resp, err := httputils.Request(httputils.GetDefaultClient(), ctx, httputils.GET, url, nil, nil, true)
	if err != nil {
		return "", errors.Wrapf(err, "request url %s", url)
	}
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return string(buf), nil
}

func (s *metadataService) fetchMetadataJSONAttr(ctx context.Context, attr string) (jsonutils.JSONObject, error) {
	version := LatestSupportedVersion

	url := s.URL.Fill(version, "meta-data", attr)
	_, resp, err := httputils.JSONRequest(httputils.GetDefaultClient(), ctx, httputils.GET, url, nil, nil, true)
	if err != nil {
		return nil, errors.Wrapf(err, "request url %s", url)
	}
	return resp, nil
}

func (s *metadataService) GetType() datasource.DatasourceType {
	return datasource.DatasourceTypeEC2
}

func (s *metadataService) FetchHostname(ctx context.Context) (string, error) {
	ret, err := s.fetchMetadataTextAttr(ctx, "hostname")
	if err != nil {
		return "", err
	}
	return ret, nil
}

func (s *metadataService) FetchLocalIPv4(ctx context.Context) (net.IP, error) {
	ret, err := s.fetchMetadataTextAttr(ctx, "local-ipv4")
	if err != nil {
		return net.IP{}, err
	}
	return net.ParseIP(ret), nil
}

func (s *metadataService) FetchPublicIPv4(ctx context.Context) (net.IP, error) {
	ret, err := s.fetchMetadataTextAttr(ctx, "public-ipv4")
	if err != nil {
		return net.IP{}, err
	}
	return net.ParseIP(ret), nil
}

func (s *metadataService) FetchMetadata(ctx context.Context) (*metadata.Digest, error) {
	hostname, err := s.FetchHostname(ctx)
	if err != nil {
		return nil, err
	}

	localIPv4, err := s.FetchLocalIPv4(ctx)
	if err != nil {
		return nil, err
	}

	publicIPv4, err := s.FetchPublicIPv4(ctx)
	if err != nil {
		return nil, err
	}

	digest := &metadata.Digest{
		Hostname:   hostname,
		LocalIPv4:  localIPv4,
		PublicIPv4: publicIPv4,
	}
	return digest, nil
}
