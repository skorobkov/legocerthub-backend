package app

import (
	"fmt"
	"net/http"
	"time"
)

const buildDir = "./frontend_build"

// runFrontend launches a web server to host the frontend
func (app *Application) runFrontend() {
	// configure webserver
	readTimeout := 5 * time.Second
	writeTimeout := 10 * time.Second
	// allow longer timeouts when in development
	if *app.config.DevMode {
		readTimeout = 15 * time.Second
		writeTimeout = 30 * time.Second
	}

	// http server config
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", *app.config.Hostname, *app.config.Frontend.HttpPort),
		IdleTimeout:  1 * time.Minute,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	// make file server handle for build dir
	http.Handle("/", http.FileServer(http.Dir(buildDir)))

	// configure and launch https if app succesfully got a cert
	if app.appCert != nil {
		// make tls config
		tlsConf, err := app.TlsConf()
		if err != nil {
			app.logger.Panicf("tls config problem: %s", err)
			return
		}

		// https server config
		srv.Addr = fmt.Sprintf("%s:%d", *app.config.Hostname, *app.config.Frontend.HttpsPort)
		srv.TLSConfig = tlsConf

		// launch https
		app.logger.Infof("starting lego-certhub frontend (https) on %s", srv.Addr)
		app.logger.Panic(srv.ListenAndServeTLS("", ""))
	} else {
		// if https failed, launch localhost only http server
		app.logger.Warnf("starting insecure lego-certhub frontend (http) on %s", srv.Addr)
		app.logger.Panic(srv.ListenAndServe())
	}
}