package cli

import (
	"github.com/shuldan/framework/pkg/contracts"
)

type cli struct {
	registry    contracts.CliRegistry
	cmdParser   *cmdParser
	cmdExecutor *cmdExecutor
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

func (c *cli) Register(cmd contracts.CliCommand) error {
	err := c.registry.Register(cmd)
	if err != nil {
		return err
	}
	return nil
}

func (c *cli) Run(ctx contracts.CliContext) error {
	return c.cmdExecutor.Execute(ctx)
}
