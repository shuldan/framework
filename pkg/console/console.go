package console

import (
	"github.com/shuldan/framework/pkg/application"
	"os"
)

type console struct {
	registry    Registry
	cmdParser   *cmdParser
	cmdExecutor *cmdExecutor
}

func (c *console) Register(cmd Command) error {
	err := c.registry.Register(cmd)
	if err != nil {
		return err
	}
	return nil
}

func (c *console) Run(appCtx application.Context) error {
	return c.cmdExecutor.Execute(
		newContext(
			appCtx,
			os.Stdin,
			os.Stdout,
			os.Args[1:],
		),
	)
}
