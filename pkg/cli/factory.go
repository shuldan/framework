package cli

import (
	"github.com/shuldan/framework/pkg/contracts"
)

func NewRegistry() contracts.CliRegistry {
	return &cmdRegistry{
		commands: make(map[string]contracts.CliCommand),
		groups:   make(map[string][]string),
	}
}

func New(registry contracts.CliRegistry) (contracts.Cli, error) {
	if registry == nil {
		registry = NewRegistry()
	}

	p := newParser(registry)
	e := newExecutor(p)

	c := &cli{
		registry:    registry,
		cmdParser:   p,
		cmdExecutor: e,
	}

	return c, nil
}

func NewModule() contracts.AppModule {
	return &module{}
}
