package http

import (
	"time"

	"github.com/shuldan/framework/pkg/contracts"
	"github.com/shuldan/framework/pkg/errors"
)

type clientModule struct{}

func NewClientModule() contracts.AppModule {
	return &clientModule{}
}

func (m *clientModule) Name() string {
	return contracts.HTTPClientModuleName
}

func (m *clientModule) Register(container contracts.DIContainer) error {
	logger, err := container.Resolve(contracts.LoggerModuleName)
	if err != nil {
		return ErrLoggerNotFound.WithCause(err)
	}
	loggerInst, ok := logger.(contracts.Logger)
	if !ok {
		return ErrInvalidLoggerInstance
	}

	return container.Factory(contracts.HTTPClientModuleName, func(c contracts.DIContainer) (interface{}, error) {
		return NewClient(loggerInst), nil
	})
}

func (m *clientModule) Start(contracts.AppContext) error {
	return nil
}

func (m *clientModule) Stop(contracts.AppContext) error {
	return nil
}

type ServerModuleOption func(*serverModule)

func WithHTTPTimeout(timeout time.Duration) ServerModuleOption {
	return func(m *serverModule) {
		m.timeout = timeout
	}
}

type serverModule struct {
	address string
	timeout time.Duration
}

func NewServerModule(address string, options ...ServerModuleOption) contracts.AppModule {
	module := &serverModule{
		address: address,
		timeout: 10 * time.Second,
	}

	for _, opt := range options {
		opt(module)
	}

	return module
}

func (m *serverModule) Name() string {
	return contracts.HTTPServerModuleName
}

func (m *serverModule) Register(container contracts.DIContainer) error {
	logger, err := container.Resolve(contracts.LoggerModuleName)
	if err != nil {
		return ErrLoggerNotFound.WithCause(err)
	}
	loggerInst, ok := logger.(contracts.Logger)
	if !ok {
		return ErrInvalidLoggerInstance
	}

	if err := container.Factory(contracts.HTTPRouterModuleName, func(c contracts.DIContainer) (interface{}, error) {
		return NewRouter(loggerInst), nil
	}); err != nil {
		return err
	}

	return container.Factory(contracts.HTTPServerModuleName, func(c contracts.DIContainer) (interface{}, error) {
		router, err := c.Resolve(contracts.HTTPRouterModuleName)
		if err != nil {
			return nil, ErrHTTPRouterNotFound.WithCause(err)
		}
		routerInst, ok := router.(contracts.HTTPRouter)
		if !ok {
			return nil, ErrInvalidHTTPRouterInstance
		}

		return NewServer(m.address, routerInst, loggerInst)
	})
}

func (m *serverModule) Start(ctx contracts.AppContext) error {
	server, err := ctx.Container().Resolve(contracts.HTTPServerModuleName)
	if err != nil {
		return ErrHTTPServerNotFound.WithCause(err)
	}
	serverInst, ok := server.(contracts.HTTPServer)
	if !ok {
		return ErrInvalidHTTPServerInstance
	}

	logger, err := ctx.Container().Resolve(contracts.LoggerModuleName)
	if err != nil {
		return ErrLoggerNotFound.WithCause(err)
	}
	loggerInst, ok := logger.(contracts.Logger)
	if !ok {
		return ErrInvalidLoggerInstance
	}

	go func() {
		if err := serverInst.Start(ctx.Ctx()); err != nil && !errors.Is(err, ctx.Ctx().Err()) {
			loggerInst.Error("HTTP server failed", "error", err)
		}
	}()

	return nil
}

func (m *serverModule) Stop(ctx contracts.AppContext) error {
	server, err := ctx.Container().Resolve(contracts.HTTPServerModuleName)
	if err != nil {
		return nil
	}
	serverInst, ok := server.(contracts.HTTPServer)
	if !ok {
		return ErrInvalidHTTPServerInstance
	}

	logger, err := ctx.Container().Resolve(contracts.LoggerModuleName)
	if err != nil {
		return ErrLoggerNotFound.WithCause(err)
	}
	loggerInst, ok := logger.(contracts.Logger)
	if !ok {
		return ErrInvalidLoggerInstance
	}

	loggerInst.Info("Stopping HTTP server...")

	return serverInst.Stop(ctx.Ctx())
}
