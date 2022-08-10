package output

import "net/http"

// WritePem sends the pem string to the client as the appropriate
// application type
func (service *Service) WritePem(w http.ResponseWriter, pem string) (bytesWritten int, err error) {
	service.logger.Debug("writing file to client")

	// for cert chain: application/pem-certificate-chain
	w.Header().Set("Content-Type", "application/x-pem-file")
	w.WriteHeader(http.StatusOK)

	bytesWritten, err = w.Write([]byte(pem))
	if err != nil {
		return -1, err
	}

	return bytesWritten, nil
}
