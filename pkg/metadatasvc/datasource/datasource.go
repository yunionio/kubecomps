package datasource

import (
	"context"
	"net"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/kubecomps/pkg/metadatasvc/metadata"
	// "yunion.io/x/kubecomps/pkg/metadatasvc/userdata"
)

const (
	ErrDatasourceRetrievalTimeout = errors.Error("datasource: timeout during data-source retrieval")
)

const (
	DatasourceTypeEC2 = "ec2"
)

var (
	providers map[DatasourceType]Provider = make(map[DatasourceType]Provider)
)

type DatasourceType string

type Provider interface {
	GetType() DatasourceType
	FetchHostname(ctx context.Context) (string, error)
	FetchLocalIPv4(ctx context.Context) (net.IP, error)
	FetchPublicIPv4(ctx context.Context) (net.IP, error)
	FetchMetadata(ctx context.Context) (*metadata.Digest, error)
	// FetchUserdata() (userdata.Map, error)
}

func RegisterProvider(p Provider) {
	_, ok := providers[p.GetType()]
	if ok {
		log.Fatalf("Provider type %s already registered", p.GetType())
	}
	providers[p.GetType()] = p
}

func GetProviders() map[DatasourceType]Provider {
	return providers
}

// FindProvider checks the given datasource providers, if it finds an available
// data source before the specified duration, it returns the provider, else it
// returns an ErrDatasourceRetrievalTimeout error
func FindProvider(providers map[DatasourceType]Provider, timeout time.Duration) (Provider, error) {
	providerCh := make(chan Provider)

	for _, provider := range providers {
		go func(p Provider) {
			if isAvailable(context.Background(), p) {
				providerCh <- p
			}
		}(provider)
	}

	timeoutChan := time.NewTicker(timeout).C

	select {
	case p := <-providerCh:
		return p, nil
	case <-timeoutChan:
		return nil, ErrDatasourceRetrievalTimeout
	}
}

func isAvailable(ctx context.Context, p Provider) bool {
	_, err := p.FetchMetadata(ctx)
	if err != nil {
		log.Warningf("Provider %s isn't available", p.GetType())
		return false
	}
	return true
}
