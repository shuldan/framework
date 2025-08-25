package contracts

import (
	"context"
	"time"
)

const (
	CliModuleName      = "cli"
	EventBusModuleName = "event.bus"
	LoggerModuleName   = "logger"
	ConfigModuleName   = "config"
)

type DIContainer interface {
	Has(name string) bool
	Instance(name string, value interface{}) error
	Factory(name string, factory func(c DIContainer) (interface{}, error)) error
	Resolve(name string) (interface{}, error)
}

type AppContext interface {
	Ctx() context.Context
	Container() DIContainer
	AppName() string
	Version() string
	Environment() string
	StartTime() time.Time
	StopTime() time.Time
	IsRunning() bool
	Stop()
}

type AppModule interface {
	Name() string
	Register(container DIContainer) error
	Start(ctx AppContext) error
	Stop(ctx AppContext) error
}

type AppRegistry interface {
	Register(module AppModule) error
	All() []AppModule
	Shutdown(ctx AppContext) error
}

type App interface {
	Register(module AppModule) error
	Run() error
}
