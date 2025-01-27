package dns01cloudflare

import (
	"errors"
	"legocerthub-backend/pkg/datatypes"

	"go.uber.org/zap"
)

var (
	errServiceComponent = errors.New("necessary dns-01 cloudflare challenge service component is missing")
	errNoDomains        = errors.New("cloudflare config error: no domains (zones) found")
)

// App interface is for connecting to the main app
type App interface {
	GetLogger() *zap.SugaredLogger
}

// Accounts service struct
type Service struct {
	logger           *zap.SugaredLogger
	knownDomainZones *datatypes.SafeMap
	dnsRecords       *datatypes.SafeMap
}

// NewService creates a new service
func NewService(app App, config *Config) (*Service, error) {
	// if disabled, return nil and no error
	if !*config.Enable {
		return nil, nil
	}

	service := new(Service)

	// logger
	service.logger = app.GetLogger()
	if service.logger == nil {
		return nil, errServiceComponent
	}

	// cloudflare api
	err := service.configureCloudflareAPI(config)
	if err != nil {
		return nil, err
	}

	// make sure at least one domain configured, or the config is bad
	if len(service.knownDomainZones.ListKeys()) <= 0 {
		return nil, errNoDomains
	}

	// debug log configured domains
	service.logger.Infof("dns01cloudflare configured domains: %s", service.knownDomainZones.ListKeys())

	// map to hold current dnsRecords
	service.dnsRecords = datatypes.NewSafeMap()

	return service, nil
}
