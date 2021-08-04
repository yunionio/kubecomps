package main

import (
	"context"

	"yunion.io/x/log"

	"yunion.io/x/kubecomps/pkg/kubeserver/app"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func main() {
	go procutils.WaitZombieLoop(context.TODO())

	if err := app.Run(context.Background()); err != nil {
		log.Fatalf("Run error: %v", err)
	}
}
