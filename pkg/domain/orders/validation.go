package orders

import (
	"errors"
	"legocerthub-backend/pkg/output"
	"legocerthub-backend/pkg/storage"
	"legocerthub-backend/pkg/validation"
	"time"
)

var (
	ErrCertIdBad  = errors.New("certificate id is invalid")
	ErrOrderIdBad = errors.New("order id is invalid")
	ErrIdMismatch = errors.New("order id does not match cert")

	ErrOrderRetryFinal      = errors.New("can't retry an order that is in a final state (valid or invalid)")
	ErrOrderRevokeBadReason = errors.New("bad revocation reason code")
)

// getOrder returns the Order specified by the ids, so long as the Order belongs
// to the Certificate.  An error is returned if the order doesn't exist or if the
// order does not belong to the cert.
func (service *Service) getOrder(certId int, orderId int) (Order, error) {
	// basic check
	if !validation.IsIdExistingValidRange(certId) {
		service.logger.Debug(ErrCertIdBad)
		return Order{}, output.ErrValidationFailed
	}
	if !validation.IsIdExistingValidRange(orderId) {
		service.logger.Debug(ErrOrderIdBad)
		return Order{}, output.ErrValidationFailed
	}

	// get order from storage
	order, err := service.storage.GetOneOrder(orderId)
	if err != nil {
		// special error case for no record found
		if err == storage.ErrNoRecord {
			service.logger.Debug(err)
			return Order{}, output.ErrNotFound
		} else {
			service.logger.Error(err)
			return Order{}, output.ErrStorageGeneric
		}
	}

	// check the cert id on the order matches the cert
	if certId != order.Certificate.ID {
		service.logger.Debug(ErrIdMismatch)
		return Order{}, output.ErrValidationFailed
	}

	return order, nil
}

// isOrderRetryable returns an error if the order is not valid, the order doesn't
// belong to the specified cert, or the order is not in a state that can be retried.
func (service *Service) isOrderRetryable(certId int, orderId int) error {
	order, err := service.getOrder(certId, orderId)
	if err != nil {
		return err
	}

	// check if order is in a final state (can't retry)
	if order.Status == "valid" || order.Status == "invalid" {
		service.logger.Debug(ErrOrderRetryFinal)
		return output.ErrValidationFailed
	}

	return nil
}

// isOrderRevocable verifies order belongs to cert and confirms the order
// is in a state that can be revoked ('valid' and 'valid_to' < current time)
func (service *Service) getOrderForRevocation(certId, orderId int) (Order, error) {
	order, err := service.getOrder(certId, orderId)
	if err != nil {
		return Order{}, err
	}

	// check order is in a state that can be revoked
	// nil check
	if order.ValidTo == nil {
		return Order{}, output.ErrValidationFailed
	}

	// confirm order is valid, not already revoked, and not expired (time)
	if !(order.Status == "valid" && !order.KnownRevoked && int(time.Now().Unix()) < *order.ValidTo) {
		return Order{}, output.ErrValidationFailed
	}

	return order, nil
}

// validRevocationReason returns an error if the specified reasonCode
// is not valid (see: rfc5280 section-5.3.1)
func (service *Service) validRevocationReason(reasonCode int) error {
	// valid codes are 0 through 10 inclusive, except 7
	if reasonCode < 0 || reasonCode == 7 || reasonCode > 10 {
		service.logger.Debug(ErrOrderRevokeBadReason)
		return output.ErrValidationFailed
	}

	return nil
}
