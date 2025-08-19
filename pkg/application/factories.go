package application

import "time"

func NewContainer() Container {
	return &container{
		factories: make(map[string]func(c Container) (interface{}, error)),
		instances: make(map[string]interface{}),
		resolving: make(map[string]bool),
	}
}

func NewRegistry() Registry {
	return &registry{
		modules: make([]Module, 0),
	}
}

func NewApplication(info AppInfo, container Container, registry Registry, opts ...func(*app)) Application {
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
