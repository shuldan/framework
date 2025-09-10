package http

import (
	"net/http"
	"time"

	"github.com/shuldan/framework/pkg/contracts"
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

type serverModule struct{}

func NewServerModule() contracts.AppModule {
	return &serverModule{}
}

func (m *serverModule) Name() string {
	return contracts.HTTPServerModuleName
}

func (m *serverModule) Register(container contracts.DIContainer) error {
	if err := registerRouter(container); err != nil {
		return err
	}

	if err := registerServer(container); err != nil {
		return err
	}

	return nil
}

func (m *serverModule) Start(ctx contracts.AppContext) error {
	routerRaw, err := ctx.Container().Resolve(contracts.HTTPRouterModuleName)
	if err != nil {
		return ErrHTTPRouterNotFound.WithCause(err)
	}
	router, ok := routerRaw.(contracts.HTTPRouter)
	if !ok {
		return ErrInvalidHTTPRouterInstance
	}

	for _, module := range ctx.AppRegistry().All() {
		if routeProvider, ok := module.(contracts.HTTPRouteProvider); ok {
			routes, err := routeProvider.HTTPRoutes(ctx)
			if err != nil {
				return ErrFailedRegisterRoutes.WithCause(err).WithDetail("module", module.Name())
			}

			for _, r := range routes {
				switch r.Method {
				case http.MethodGet:
					router.GET(r.Path, r.Handler, r.Middleware...)
				case http.MethodPost:
					router.POST(r.Path, r.Handler, r.Middleware...)
				case http.MethodPut:
					router.PUT(r.Path, r.Handler, r.Middleware...)
				case http.MethodDelete:
					router.DELETE(r.Path, r.Handler, r.Middleware...)
				case http.MethodPatch:
					router.PATCH(r.Path, r.Handler, r.Middleware...)
				case http.MethodHead:
					router.HEAD(r.Path, r.Handler, r.Middleware...)
				case http.MethodOptions:
					router.OPTIONS(r.Path, r.Handler, r.Middleware...)
				default:
					router.Handle(r.Method, r.Path, r.Handler, r.Middleware...)
				}
			}
		}
	}

	return nil
}

func (m *serverModule) Stop(ctx contracts.AppContext) error {
	return nil
}

func (m *serverModule) CliCommands(ctx contracts.AppContext) ([]contracts.CliCommand, error) {
	logger, err := ctx.Container().Resolve(contracts.LoggerModuleName)
	if err != nil {
		return nil, ErrLoggerNotFound.WithCause(err)
	}
	loggerInst, ok := logger.(contracts.Logger)
	if !ok {
		return nil, ErrInvalidLoggerInstance
	}

	router, err := ctx.Container().Resolve(contracts.HTTPRouterModuleName)
	if err != nil {
		return nil, ErrHTTPRouterNotFound.WithCause(err)
	}
	routerInst, ok := router.(contracts.HTTPRouter)
	if !ok {
		return nil, ErrInvalidHTTPRouterInstance
	}

	config, err := ctx.Container().Resolve(contracts.ConfigModuleName)
	if err != nil {
		return nil, ErrConfigNotFound.WithCause(err)
	}
	configInst, ok := config.(contracts.Config)
	if !ok {
		return nil, ErrInvalidConfigInstance
	}

	server, err := ctx.Container().Resolve(contracts.HTTPServerModuleName)
	if err != nil {
		return nil, ErrHTTPServerNotFound.WithCause(err)
	}
	serverInst, ok := server.(contracts.HTTPServer)
	if !ok {
		return nil, ErrInvalidHTTPServerInstance
	}

	cliCommands := []contracts.CliCommand{
		NewServerCommand(configInst, loggerInst, serverInst, routerInst),
	}

	return cliCommands, nil
}

func registerRouter(container contracts.DIContainer) error {
	logger, err := container.Resolve(contracts.LoggerModuleName)
	if err != nil {
		return ErrLoggerNotFound.WithCause(err)
	}
	loggerInst, ok := logger.(contracts.Logger)
	if !ok {
		return ErrInvalidLoggerInstance
	}

	config, err := container.Resolve(contracts.ConfigModuleName)
	if err != nil {
		return ErrConfigNotFound.WithCause(err)
	}
	configInst, ok := config.(contracts.Config)
	if !ok {
		return ErrInvalidConfigInstance
	}

	router := NewRouter(loggerInst)
	middlewares := LoadMiddlewareFromConfig(configInst, loggerInst)
	for _, mw := range middlewares {
		router.Use(mw)
	}

	return container.Factory(contracts.HTTPRouterModuleName, func(c contracts.DIContainer) (interface{}, error) {
		return router, nil
	})
}

func registerServer(container contracts.DIContainer) error {
	logger, err := container.Resolve(contracts.LoggerModuleName)
	if err != nil {
		return ErrLoggerNotFound.WithCause(err)
	}
	loggerInst, ok := logger.(contracts.Logger)
	if !ok {
		return ErrInvalidLoggerInstance
	}

	router, err := container.Resolve(contracts.HTTPRouterModuleName)
	if err != nil {
		return ErrHTTPRouterNotFound.WithCause(err)
	}
	routerInst, ok := router.(contracts.HTTPRouter)
	if !ok {
		return ErrInvalidHTTPRouterInstance
	}

	var options []ServerOption

	if config, err := container.Resolve(contracts.ConfigModuleName); err == nil {
		if cfg, ok := config.(contracts.Config); ok {
			if httpCfg, ok := cfg.GetSub("http.server"); ok {
				options = append(options,
					WithAddress(httpCfg.GetString("address", ":8080")),
					WithReadHeaderTimeout(time.Duration(httpCfg.GetInt("read_header_timeout", 30))*time.Second),
					WithReadTimeout(time.Duration(httpCfg.GetInt("read_timeout", 30))*time.Second),
					WithWriteTimeout(time.Duration(httpCfg.GetInt("write_timeout", 60))*time.Second),
					WithIdleTimeout(time.Duration(httpCfg.GetInt("idle_timeout", 90))*time.Second),
					WithShutdownTimeout(time.Duration(httpCfg.GetInt("shutdown_timeout", 30))*time.Second),
				)
			}
		}
	}

	return container.Factory(contracts.HTTPServerModuleName, func(c contracts.DIContainer) (interface{}, error) {
		return NewServer(routerInst, loggerInst, options...)
	})
}
