package cli

import (
	"github.com/shuldan/framework/pkg/contracts"
	"os"
)

type cli struct {
	registry    contracts.CliRegistry
	cmdParser   *cmdParser
	cmdExecutor *cmdExecutor
}

func (c *cli) Register(cmd contracts.CliCommand) error {
	err := c.registry.Register(cmd)
	if err != nil {
		return err
	}
	return nil
}

func (c *cli) Run(appCtx contracts.AppContext) error {
	return c.cmdExecutor.Execute(
		newContext(
			appCtx,
			os.Stdin,
			os.Stdout,
			os.Args[1:],
		),
	)
}
