package orders

import (
	"context"
	"database/sql"
	"errors"
	"legocerthub-backend/pkg/randomness"
	"sync"
	"time"
)

// startAutoOrderService starts a go routine that completes existing orders that are
// not yet in a 'valid' or 'invalid' state and also places new orders forexpiring certs
// The service runs daily at the time specified in consts.
func (service *Service) startAutoOrderService(cfg *Config, ctx context.Context, wg *sync.WaitGroup) {
	// dont run if not enabled
	if !*cfg.AutomaticOrderingEnable {
		return
	}

	// calculate timing based on config
	remainingDaysThreshold := time.Duration(*cfg.ValidRemainingDaysThreshold) * (24 * time.Hour)
	refreshHour := *cfg.RefreshTimeHour
	refreshMinute := *cfg.RefreshTimeMinute

	// log start and update wg
	service.logger.Infof("starting automatic certificate ordering service; %d day expiration threshold; "+
		"orders will be placed every day at %d:%d", *cfg.ValidRemainingDaysThreshold, refreshHour, refreshMinute)
	wg.Add(1)

	// service routine
	go func() {
		defer wg.Done()
		var nextRunTime time.Time

		// indefinite service loop
		for {
			// run time for today
			nextRunTime = time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(),
				refreshHour, refreshMinute, 0, 0, time.Local)

			// if today's run already passed, run tomorrow
			if !nextRunTime.After(time.Now()) {
				nextRunTime = nextRunTime.Add(24 * time.Hour)
			}

			// add random second to runtime, as preferred by Let's Encrypt
			// see: https://letsencrypt.org/docs/integration-guide/#when-to-renew
			// added after timestamp calc to avoid accidental duplicate run on same day
			// e.g. if runs at :12 and then next timestamp is :50, it is possible for the
			// new stamp to not be after now and therefore would run a second time
			refreshSecond, err := randomness.GenerateRandomInt(60)
			if err != nil {
				// if error, use 12
				service.logger.Errorf("failed to generate auto order random second integer (%s)", err)
				refreshSecond = 12
			}
			nextRunTime = nextRunTime.Add(time.Duration(refreshSecond) * time.Second)

			// sleep or wait for shutdown context to be done
			select {
			case <-ctx.Done():
				// close routine
				service.logger.Info("automatic certificate ordering service shutdown complete")
				return

			case <-time.After(time.Until(nextRunTime)):
				// sleep until run time
			}

			// complete existing orders that are not 'valid' or 'invalid' (i.e. not completed)
			err = service.retryIncompleteOrders()
			if err != nil {
				service.logger.Errorf("error retying incomplete orders: %s", err)
			}

			// order expiring certificates
			err = service.orderExpiringCerts(remainingDaysThreshold)
			if err != nil {
				service.logger.Errorf("error ordering expiring certs: %s", err)
			}
		}
	}()
}

// retryIncompleteOrders retries all incomplete orders within storage. this should
// move all orders to valid or invalid state.
func (service *Service) retryIncompleteOrders() (err error) {
	service.logger.Info("adding incomplete orders to order queue")

	// get all incomplete order ids from storage
	incompleteOrderIds, err := service.storage.GetAllIncompleteOrderIds()
	if err != nil {
		return err
	}

	// add all incompletes to the low priority order queue
	for _, orderId := range incompleteOrderIds {
		err = service.orderFromAcme(orderId, false)
		if err != nil {
			// log error, but keep going through remaining range
			service.logger.Errorf("failed to add order %d to processing queue (%s)", orderId, err)
		}
	}

	service.logger.Info("incomplete orders added to order queue")
	return nil
}

// orderExpiringCerts automatically orders any certficates that are valid but have a valid_to
// timestamp within the specified threshold
func (service *Service) orderExpiringCerts(remainingDaysThreshold time.Duration) (err error) {
	service.logger.Info("adding expiring certificates to order queue")

	// get slice of all expiring certificate ids
	expiringCertIds, err := service.storage.GetExpiringCertIds(remainingDaysThreshold)
	if err != nil {
		return err
	}

	// address each expiring cert
	for _, certId := range expiringCertIds {
		// check for an existing incomplete order
		orderId, err := service.storage.GetNewestIncompleteCertOrderId(certId)

		if err != nil {
			// unable to get existing incomplete order -> place new order
			// if error other than NoRows, log it
			if err != sql.ErrNoRows {
				service.logger.Errorf("failed to fetch newest incomplete order id for cert %d (%s)", certId, err)
			}

			// place new order
			service.logger.Debugf("placing new order for expiring cert %d", certId)
			_, err = service.placeNewOrderAndFulfill(certId, false)
			if err != nil {
				service.logger.Errorf("failed to place new order for cert %d (%s)", certId, err)
			}

		} else {
			// no error, retry existing order
			service.logger.Debugf("retrying order %d to refresh cert %d", orderId, certId)
			err = service.orderFromAcme(orderId, false)
			if err != nil {
				service.logger.Errorf("failed to retry order %d for cert %d (%s)", orderId, certId, err)
			}
		}

		// sleep a little so slew of new orders don't hit ACME all at once
		// cancel on shutdown context
		select {
		case <-service.shutdownContext.Done():
			// abort refreshing due to shutdown
			return errors.New("expiring certificates refresh canceled due to shutdown")

		case <-time.After(15 * time.Second):
			// sleep and continue
		}
	}

	service.logger.Info("expiring certificates added to order queue")
	return nil
}
