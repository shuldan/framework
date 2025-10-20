package http

import (
	"flag"

	"github.com/shuldan/framework/pkg/contracts"
)

type serverCommand struct {
	config     contracts.Config
	logger     contracts.Logger
	httpServer contracts.HTTPServer
	httpRouter contracts.HTTPRouter
}

func NewServerCommand(
	config contracts.Config,
	logger contracts.Logger,
	httpServer contracts.HTTPServer,
	httpRouter contracts.HTTPRouter,
) contracts.CliCommand {
	return &serverCommand{
		config:     config,
		logger:     logger,
		httpServer: httpServer,
		httpRouter: httpRouter,
	}
}

func (c *serverCommand) Name() string {
	return "http:serve"
}

func (c *serverCommand) Description() string {
	return "Start the HTTP server"
}

func (c *serverCommand) Group() string {
	return contracts.HttpCliGroup
}

func (c *serverCommand) Configure(flags *flag.FlagSet) {}

func (c *serverCommand) Validate(ctx contracts.CliContext) error {
	return nil
}

func (c *serverCommand) Execute(ctx contracts.CliContext) error {
	serverErr := make(chan error, 1)
	go func() {
		err := c.httpServer.Start(ctx.AppContext().ParentContext())
		serverErr <- err
	}()

	select {
	case <-ctx.AppContext().ParentContext().Done():
		c.logger.Info("Shutting down HTTP server...")
		if err := c.httpServer.Stop(ctx.AppContext().ParentContext()); err != nil {
			c.logger.Error("Failed to stop HTTP server", "error", err)
			return err
		}
		c.logger.Info("HTTP server stopped gracefully")
	case err := <-serverErr:
		if err != nil {
			c.logger.Error("HTTP server failed", "error", err)
			return err
		}
	}

	return nil
}
