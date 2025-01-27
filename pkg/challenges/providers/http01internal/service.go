package http01internal

import (
	"context"
	"errors"
	"sync"

	"go.uber.org/zap"
)

var (
	errServiceComponent = errors.New("necessary http-01 internal challenge service component is missing")
	errConfigComponent  = errors.New("necessary http-01 config option missing")
)

// App interface is for connecting to the main app
type App interface {
	GetDevMode() bool
	GetLogger() *zap.SugaredLogger
	GetShutdownContext() context.Context
	GetShutdownWaitGroup() *sync.WaitGroup
}

// Accounts service struct
type Service struct {
	devMode bool
	logger  *zap.SugaredLogger
	tokens  map[string]string
	mu      sync.RWMutex // added mutex due to unsafe if add and remove token both run
}

// Configuration options
type Config struct {
	Enable *bool `yaml:"enable"`
	Port   *int  `yaml:"port"`
}

// NewService creates a new service
func NewService(app App, config *Config) (*Service, error) {
	// if disabled, return nil and no error
	if !*config.Enable {
		return nil, nil
	}

	service := new(Service)

	// devmode?
	service.devMode = app.GetDevMode()

	// logger
	service.logger = app.GetLogger()
	if service.logger == nil {
		return nil, errServiceComponent
	}

	// allocate token map
	service.tokens = make(map[string]string, 50)

	// start web server for http01 challenges
	if config.Port == nil {
		return nil, errConfigComponent
	}
	err := service.startServer(*config.Port, app.GetShutdownContext(), app.GetShutdownWaitGroup())
	if err != nil {
		return nil, err
	}

	return service, nil
}
