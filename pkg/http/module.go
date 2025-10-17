package http

import (
	"net/http"
	"reflect"
	"time"

	"github.com/shuldan/framework/pkg/contracts"
)

const (
	ModuleClientName = "http.client"
	ModuleServerName = "http.server"
)

type clientModule struct{}

func NewClientModule() contracts.AppModule {
	return &clientModule{}
}

func (m *clientModule) Name() string {
	return ModuleClientName
}

func (m *clientModule) Register(container contracts.DIContainer) error {
	logger, err := container.Resolve(reflect.TypeOf((*contracts.Logger)(nil)).Elem())
	if err != nil {
		return ErrLoggerNotFound.WithCause(err)
	}
	loggerInst, ok := logger.(contracts.Logger)
	if !ok {
		return ErrInvalidLoggerInstance
	}

	return container.Factory(reflect.TypeOf((*contracts.HTTPClient)(nil)).Elem(), func(c contracts.DIContainer) (interface{}, error) {
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
	return ModuleServerName
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
	routerRaw, err := ctx.Container().Resolve(reflect.TypeOf((*contracts.HTTPRouter)(nil)).Elem())
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
	logger, err := ctx.Container().Resolve(reflect.TypeOf((*contracts.Logger)(nil)).Elem())
	if err != nil {
		return nil, ErrLoggerNotFound.WithCause(err)
	}
	loggerInst, ok := logger.(contracts.Logger)
	if !ok {
		return nil, ErrInvalidLoggerInstance
	}

	router, err := ctx.Container().Resolve(reflect.TypeOf((*contracts.HTTPRouter)(nil)).Elem())
	if err != nil {
		return nil, ErrHTTPRouterNotFound.WithCause(err)
	}
	routerInst, ok := router.(contracts.HTTPRouter)
	if !ok {
		return nil, ErrInvalidHTTPRouterInstance
	}

	config, err := ctx.Container().Resolve(reflect.TypeOf((*contracts.Config)(nil)).Elem())
	if err != nil {
		return nil, ErrConfigNotFound.WithCause(err)
	}
	configInst, ok := config.(contracts.Config)
	if !ok {
		return nil, ErrInvalidConfigInstance
	}

	server, err := ctx.Container().Resolve(reflect.TypeOf((*contracts.HTTPServer)(nil)).Elem())
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
	logger, err := container.Resolve(reflect.TypeOf((*contracts.Logger)(nil)).Elem())
	if err != nil {
		return ErrLoggerNotFound.WithCause(err)
	}
	loggerInst, ok := logger.(contracts.Logger)
	if !ok {
		return ErrInvalidLoggerInstance
	}

	config, err := container.Resolve(reflect.TypeOf((*contracts.Config)(nil)).Elem())
	if err != nil {
		return ErrConfigNotFound.WithCause(err)
	}
	configInst, ok := config.(contracts.Config)
	if !ok {
		return ErrInvalidConfigInstance
	}

	router := NewRouter(loggerInst)
	middlewares := LoadMiddlewareFromConfig(configInst, loggerInst)
	router.Use(middlewares...)

	return container.Instance(reflect.TypeOf((*contracts.HTTPRouter)(nil)).Elem(), router)
}

func registerServer(container contracts.DIContainer) error {
	logger, err := container.Resolve(reflect.TypeOf((*contracts.Logger)(nil)).Elem())
	if err != nil {
		return ErrLoggerNotFound.WithCause(err)
	}
	loggerInst, ok := logger.(contracts.Logger)
	if !ok {
		return ErrInvalidLoggerInstance
	}

	router, err := container.Resolve(reflect.TypeOf((*contracts.HTTPRouter)(nil)).Elem())
	if err != nil {
		return ErrHTTPRouterNotFound.WithCause(err)
	}
	routerInst, ok := router.(contracts.HTTPRouter)
	if !ok {
		return ErrInvalidHTTPRouterInstance
	}

	var options []ServerOption

	if config, err := container.Resolve(reflect.TypeOf((*contracts.Config)(nil)).Elem()); err == nil {
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

	server, err := NewServer(routerInst, loggerInst, options...)
	if err != nil {
		return err
	}

	return container.Instance(reflect.TypeOf((*contracts.HTTPServer)(nil)).Elem(), server)
}
