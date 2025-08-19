package application

import (
	"context"
	"time"
)

type Container interface {
	Has(name string) bool
	Instance(name string, value interface{}) error
	Factory(name string, factory func(c Container) (interface{}, error)) error
	Resolve(name string) (interface{}, error)
}

type Context interface {
	Ctx() context.Context
	Container() Container
	AppName() string
	Version() string
	Environment() string
	StartTime() time.Time
	StopTime() time.Time
	IsRunning() bool
	Stop()
}

type Module interface {
	Name() string
	Register(container Container) error
	Start(ctx Context) error
	Stop(ctx Context) error
}

type Registry interface {
	Register(module Module) error
	All() []Module
	Shutdown(ctx Context) error
}

type Application interface {
	Register(module Module) error
	Run() error
}
