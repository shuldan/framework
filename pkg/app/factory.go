package app

import (
	"github.com/shuldan/framework/pkg/contracts"
	"time"
)

func NewContainer() contracts.DIContainer {
	return &container{
		factories: make(map[string]func(c contracts.DIContainer) (interface{}, error)),
		instances: make(map[string]interface{}),
		resolving: make(map[string]bool),
	}
}

func NewRegistry() contracts.AppRegistry {
	return &registry{
		modules: make([]contracts.AppModule, 0),
	}
}

func New(info AppInfo, container contracts.DIContainer, registry contracts.AppRegistry, opts ...func(*app)) contracts.App {
	if container == nil {
		container = NewContainer()
	}

	if registry == nil {
		registry = NewRegistry()
	}

	a := &app{
		container:       container,
		registry:        registry,
		info:            info,
		shutdownTimeout: 10 * time.Second,
	}

	for _, opt := range opts {
		opt(a)
	}

	return a
}
