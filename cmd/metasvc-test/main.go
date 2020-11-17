package main

import (
	"context"
	"time"

	"yunion.io/x/log"

	"yunion.io/x/kubecomps/pkg/metadatasvc/datasource"
	_ "yunion.io/x/kubecomps/pkg/metadatasvc/datasource/init"
)

func main() {
	provider, err := datasource.FindProvider(datasource.GetProviders(), 5*time.Second)
	if err != nil {
		log.Fatalf("FindProvider error: %v", err)
	}

	meta, err := provider.FetchMetadata(context.Background())
	if err != nil {
		log.Fatalf("Datasource %s FetchMetadata error: %v", provider.GetType(), err)
	}

	log.Infof("Meta: \n%s", meta.PrettyString())
	log.Infof("LocalIPv4: %s", meta.LocalIPv4.To4().String())
	log.Infof("PublicIPv4: %s", meta.PublicIPv4.To4().String())
}
