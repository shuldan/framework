package contracts

import (
	"context"
	"reflect"
	"time"
)

type DIContainer interface {
	Has(abstract reflect.Type) bool
	Instance(abstract reflect.Type, concrete any) error
	Factory(abstract reflect.Type, factory func(c DIContainer) (any, error)) error
	Resolve(abstract reflect.Type) (any, error)
}

type AppContext interface {
	ParentContext() context.Context
	Container() DIContainer
	AppName() string
	Version() string
	Environment() string
	StartTime() time.Time
	StopTime() time.Time
	IsRunning() bool
	Stop()
	AppRegistry() AppRegistry
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
