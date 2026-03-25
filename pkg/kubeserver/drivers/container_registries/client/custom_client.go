package client

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/kubecomps/pkg/kubeserver/api"
)

type customClient struct{}

func NewCustomClient() Client {
	return &customClient{}
}

func (c *customClient) Ping(ctx context.Context) error {
	return nil
}

func (c *customClient) ListImages(ctx context.Context, input *api.ContainerRegistryListImagesInput) (jsonutils.JSONObject, error) {
	return nil, nil
}

func (c *customClient) ListImageTags(ctx context.Context, image string) (jsonutils.JSONObject, error) {
	return nil, nil
}

func (c *customClient) AnalysisImageTarMetadata(tarPath string) (*ImageMetadata, error) {
	return nil, nil
}

func (c *customClient) PushImage(ctx context.Context, input *ImageMetadata, tarPath string) error {
	return nil
}
