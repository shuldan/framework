package bootstrap

import (
	"os"
	"time"

	"github.com/shuldan/framework/pkg/app"
	"github.com/shuldan/framework/pkg/cli"
	"github.com/shuldan/framework/pkg/config"
	"github.com/shuldan/framework/pkg/contracts"
	"github.com/shuldan/framework/pkg/database"
	"github.com/shuldan/framework/pkg/events"
	"github.com/shuldan/framework/pkg/http"
	"github.com/shuldan/framework/pkg/logger"
	"github.com/shuldan/framework/pkg/queue/broker"
)

type Bootstrap struct {
	appName         string
	appVersion      string
	appEnvironment  string
	modules         []contracts.AppModule
	gracefulTimeout time.Duration
}

func New(appName string, appVersion string, envPrefix string, configPaths ...string) *Bootstrap {
	appEnvironment := os.Getenv("APP_ENVIRONMENT")
	if appEnvironment == "" {
		appEnvironment = "development"
	}

	configModule := config.NewModule(envPrefix, configPaths...)

	modules := []contracts.AppModule{
		configModule,
	}

	return &Bootstrap{
		appName:         appName,
		appVersion:      appVersion,
		appEnvironment:  appEnvironment,
		modules:         modules,
		gracefulTimeout: 30 * time.Second,
	}
}

func (b *Bootstrap) WithGracefulTimeout(timeout time.Duration) {
	b.gracefulTimeout = timeout
}

func (b *Bootstrap) WithCli() *Bootstrap {
	m := cli.NewModule()
	b.modules = append(b.modules, m)
	return b
}

func (b *Bootstrap) WithDatabase() *Bootstrap {
	m := database.NewModule()
	b.modules = append(b.modules, m)
	return b
}

func (b *Bootstrap) WithEventBus() *Bootstrap {
	m := events.NewModule()
	b.modules = append(b.modules, m)
	return b
}

func (b *Bootstrap) WithLogger() *Bootstrap {
	m := logger.NewModule()
	b.modules = append(b.modules, m)
	return b
}

func (b *Bootstrap) WithHTTPClient() *Bootstrap {
	m := http.NewClientModule()
	b.modules = append(b.modules, m)
	return b
}

func (b *Bootstrap) WithHTTPServer() *Bootstrap {
	m := http.NewServerModule()
	b.modules = append(b.modules, m)
	return b
}

func (b *Bootstrap) WithQueueBrock() *Bootstrap {
	m := broker.NewModule()
	b.modules = append(b.modules, m)
	return b
}

func (b *Bootstrap) CreateApp() (contracts.App, error) {
	a := app.New(
		app.Info{
			AppName:     b.appName,
			Version:     b.appVersion,
			Environment: b.appEnvironment,
		},
		app.NewContainer(),
		app.NewRegistry(),
		app.WithGracefulTimeout(b.gracefulTimeout),
	)

	for _, module := range b.modules {
		if err := a.Register(module); err != nil {
			return nil, err
		}
	}

	return a, nil
}
