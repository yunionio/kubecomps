package server

import (
	"crypto/tls"
	"fmt"
	"io"
	olog "log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

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

	utilruntime.ReallyCrash = false
	serveHTTPS := func() error {
		cipherSuites := []uint16{}
		for _, suite := range tls.CipherSuites() {
			if !strings.HasSuffix(suite.Name, "_SHA") {
				cipherSuites = append(cipherSuites, suite.ID)
			}
		}

		minTLSVer := uint16(tls.VersionTLS12)
		tlsConf := &tls.Config{
			CipherSuites: cipherSuites,
			MinVersion:   minTLSVer,
		}

		s := &http.Server{
			Addr:              httpsAddr,
			Handler:           root,
			IdleTimeout:       appsrv.DEFAULT_IDLE_TIMEOUT,
			ReadTimeout:       appsrv.DEFAULT_READ_TIMEOUT,
			ReadHeaderTimeout: appsrv.DEFAULT_READ_HEADER_TIMEOUT,
			WriteTimeout:      appsrv.DEFAULT_WRITE_TIMEOUT,
			MaxHeaderBytes:    1 << 20,
			// fix aliyun elb healt check tls error
			// issue like: https://github.com/megaease/easegress/issues/481
			ErrorLog: olog.New(io.Discard, "", olog.LstdFlags),

			TLSConfig: tlsConf,
		}
		return s.ListenAndServeTLS(tlsCertFile, tlsPrivateKey)
	}
	return serveHTTPS()
}
