package server

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appsrv"

	"yunion.io/x/kubecomps/pkg/kubeserver/options"
)

func Start(httpsAddr string, app *appsrv.Application) error {
	log.Infof("Start listen on https addr: %q", httpsAddr)

	opt := options.Options

	tlsCertFile := opt.TlsCertFile
	tlsPrivateKey := opt.TlsPrivateKeyFile
	if tlsCertFile == "" || tlsPrivateKey == "" {
		return fmt.Errorf("Please specify --tls-cert-file and --tls-private-key-file")
	}

	root := mux.NewRouter()
	root.UseEncodedPath()

	httpRoot := mux.NewRouter()
	httpRoot.UseEncodedPath()

	root.PathPrefix("/api/").Handler(app)

	serveHTTPS := func() error {
		return http.ListenAndServeTLS(httpsAddr, tlsCertFile, tlsPrivateKey, root)
	}
	return serveHTTPS()
}
