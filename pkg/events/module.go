package events

import (
	"reflect"

	"github.com/shuldan/framework/pkg/contracts"
)

const ModuleName = "events"

type module struct{}

func NewModule() contracts.AppModule {
	return &module{}
}

func (m *module) Name() string {
	return ModuleName
}

func (m *module) Register(container contracts.DIContainer) error {
	return container.Factory(
		reflect.TypeOf((*contracts.EventBus)(nil)).Elem(),
		func(c contracts.DIContainer) (interface{}, error) {
			logger, err := c.Resolve(reflect.TypeOf((*contracts.Logger)(nil)).Elem())
			if err != nil {
				return nil, ErrLoggerNotFound.WithCause(err)
			}
			if logger == nil {
				return nil, ErrLoggerRequired
			}
			loggerInst, ok := logger.(contracts.Logger)
			if !ok {
				return nil, ErrInvalidLoggerInstance
			}

			options := m.getEventBusOptions(c, loggerInst)

			b := New(options...)
			return b, nil
		},
	)
}

func (m *module) Start(_ contracts.AppContext) error {
	return nil
}

func (m *module) Stop(ctx contracts.AppContext) error {
	b, err := ctx.Container().Resolve(reflect.TypeOf((*contracts.EventBus)(nil)).Elem())
	if err != nil {
		return ErrBusNotFound.WithCause(err)
	}

	busInst, ok := b.(contracts.EventBus)
	if !ok {
		return ErrInvalidBusInstance
	}

	return busInst.Close()
}

func (m *module) getEventBusOptions(c contracts.DIContainer, logger contracts.Logger) []Option {
	var options []Option

	if logger != nil {
		options = append(options,
			WithPanicHandler(NewDefaultPanicHandler(logger)),
			WithErrorHandler(NewDefaultErrorHandler(logger)),
		)
	} else {
		options = append(options,
			WithPanicHandler(NewDefaultPanicHandler(nil)),
			WithErrorHandler(NewDefaultErrorHandler(nil)),
		)
	}

	if configInst, err := c.Resolve(reflect.TypeOf((*contracts.Config)(nil)).Elem()); err == nil {
		if cfg, ok := configInst.(contracts.Config); ok {
			if eventsCfg, ok := cfg.GetSub("events"); ok {
				asyncMode := eventsCfg.GetBool("async_mode", false)
				workerCount := eventsCfg.GetInt("worker_count", 1)

				options = append(options,
					WithAsyncMode(asyncMode),
					WithWorkerCount(workerCount),
				)
			}
		}
	}

	return options
}
