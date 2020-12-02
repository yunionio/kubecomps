package s3

import (
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/s3utils"

	"yunion.io/x/pkg/errors"
)

type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Secure    bool
}

type Client struct {
	*minio.Client
}

func NewClient(conf *Config) (*Client, error) {
	cli, err := minio.New(conf.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(conf.AccessKey, conf.SecretKey, ""),
		Secure: conf.Secure,
	})
	if err != nil {
		return nil, errors.Wrap(err, "New client")
	}
	return &Client{
		Client: cli,
	}, nil
}

func CheckValidBucketNameStrict(name string) error {
	return s3utils.CheckValidBucketNameStrict(name)
}
