package app

import (
	"legocerthub-backend/acme_accounts"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *Application) Routes() http.Handler {
	router := httprouter.New()

	// app handlers
	router.HandlerFunc(http.MethodGet, "/status", app.statusHandler)

	// acme accounts handlers
	// acme accounts database definition
	acmeAccounts := acme_accounts.AcmeAccounts{
		DB:     app.DB,
		Logger: app.Logger,
	}
	router.HandlerFunc(http.MethodGet, "/v1/acmeaccounts", acmeAccounts.GetAllAcmeAccounts)
	router.HandlerFunc(http.MethodGet, "/v1/acmeaccounts/:id", acmeAccounts.GetOneAcmeAccount)

	return app.enableCORS(router)
}
