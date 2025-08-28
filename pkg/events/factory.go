package events

import (
	"reflect"

	"github.com/shuldan/framework/pkg/contracts"
)

func NewDefaultPanicHandler(logger contracts.Logger) PanicHandler {
	return &defaultPanicHandler{logger: logger}
}

func NewDefaultErrorHandler(logger contracts.Logger) ErrorHandler {
	return &defaultErrorHandler{logger: logger}
}

func New(opts ...Option) contracts.Bus {
	cfg := &config{
		asyncMode:   false,
		workerCount: 1,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.panicHandler == nil {
		cfg.panicHandler = NewDefaultPanicHandler(nil)
	}
	if cfg.errorHandler == nil {
		cfg.errorHandler = NewDefaultErrorHandler(nil)
	}

	b := &bus{
		listeners:    make(map[reflect.Type][]*listenerAdapter),
		panicHandler: cfg.panicHandler,
		errorHandler: cfg.errorHandler,
		asyncMode:    cfg.asyncMode,
		workerCount:  cfg.workerCount,
	}

	if cfg.asyncMode {
		b.startWorkers()
	}

	return b
}

func NewModule() contracts.AppModule {
	return &module{}
}
