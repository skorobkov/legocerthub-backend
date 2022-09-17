package download

import (
	"legocerthub-backend/pkg/output"
	"log"
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
)

// DownloadPrivateCertViaHeader is the handler to write just a
// cert's chain to the client, if the proper apiKey is provided via
// header (standard method)
func (service *Service) DownloadPrivateCertViaHeader(w http.ResponseWriter, r *http.Request) (err error) {
	// get cert name
	params := httprouter.ParamsFromContext(r.Context())
	keyName := params.ByName("name")

	// get apiKey from header
	apiKey := r.Header.Get("X-API-Key")
	// try to get from apikey header if X-API-Key was empty
	if apiKey == "" {
		apiKey = r.Header.Get("apikey")
	}

	// fetch the private cert
	certPem, err := service.getPrivateCertPem(keyName, apiKey, false)
	if err != nil {
		return err
	}

	// return pem file to client
	_, err = service.output.WritePem(w, certPem)
	if err != nil {
		service.logger.Error(err)
		return output.ErrWritePemFailed
	}

	return nil
}

// DownloadPrivateCertViaUrl is the handler to write just a
// cert's chain to the client, if the proper apiKey is provided via
// URL (NOT recommended - only implemented to support clients that
// can't specify the apiKey header)
func (service *Service) DownloadPrivateCertViaUrl(w http.ResponseWriter, r *http.Request) (err error) {
	// get cert name & apiKey
	params := httprouter.ParamsFromContext(r.Context())
	keyName := params.ByName("name")
	apiKey := params.ByName("apiKey")

	// fetch the private cert
	certPem, err := service.getPrivateCertPem(keyName, apiKey, true)
	if err != nil {
		return err
	}

	// return pem file to client
	_, err = service.output.WritePem(w, certPem)
	if err != nil {
		service.logger.Error(err)
		return output.ErrWritePemFailed
	}

	return nil
}

// getPrivateCertPem returns the cert's private key pem appended to the cert's
// public certificate pem. ApiKeys should be the certificate apikey appended
// to the private key's apikey using a '.' as a separator. It also checks
// the apiKeyViaUrl property if the client is making a request with the apiKey
// in the Url. The pem is from the most recent valid order for the specified cert.
// The key is the matching key for the order. An order is returned if the key
// has been deleted.
// TODO: Allow entire cert chain to be provided
func (service *Service) getPrivateCertPem(certName string, apiKeysString string, apiKeyViaUrl bool) (privateCertPem string, err error) {
	// if not running https, error
	if !service.https && !service.devMode {
		return "", output.ErrUnavailableHttp
	}

	// separate the apiKeys
	apiKeys := strings.Split(apiKeysString, ".")

	// error if not exactly 2 apiKeys
	if len(apiKeys) != 2 {
		return "", output.ErrUnauthorized
	}

	log.Println("made it")
	certApiKey := apiKeys[0]
	keyApiKey := apiKeys[1]

	// fetch the full certificate chain
	certPem, keyName, err := service.getCertPem(certName, certApiKey, false, apiKeyViaUrl)
	if err != nil {
		return "", err
	}

	// if key name is nil (key deleted), error
	if keyName == nil {
		return "", output.ErrStorageGeneric
	}

	// fetch the matching private key
	keyPem, err := service.getKeyPem(*keyName, keyApiKey, apiKeyViaUrl)
	if err != nil {
		return "", err
	}

	// append key and cert
	privateCertPem = keyPem + string([]byte{10}) + certPem

	// return pem content
	return privateCertPem, nil
}