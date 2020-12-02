package main

import (
	"context"

	"yunion.io/x/log"

	"yunion.io/x/kubecomps/pkg/kubeserver/app"
)

func main() {
	if err := app.Run(context.Background()); err != nil {
		log.Fatalf("Run error: %v", err)
	}
}
