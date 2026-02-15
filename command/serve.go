package command

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/shuldan/app"
	"github.com/shuldan/cli"

	"github.com/shuldan/framework/logger"
)

func Serve(
	name string,
	log *logger.Logger,
	timeout time.Duration,
	modules ...app.Module,
) cli.Command {
	return &serveCommand{
		appName: name,
		log:     log,
		timeout: timeout,
		modules: modules,
	}
}

type serveCommand struct {
	appName string
	log     *logger.Logger
	timeout time.Duration
	modules []app.Module
}

func (c *serveCommand) Name() string          { return "serve" }
func (c *serveCommand) Description() string   { return "Start HTTP server and background workers" }
func (c *serveCommand) Group() string         { return "server" }
func (c *serveCommand) Args() []cli.Arg       { return nil }
func (c *serveCommand) Options() []cli.Option { return nil }

func (c *serveCommand) Execute(
	ctx context.Context, _ io.Reader, _ io.Writer, _ *cli.Input,
) error {
	application, err := c.buildApp()
	if err != nil {
		return fmt.Errorf("serve: build app: %w", err)
	}

	return application.Run(ctx)
}

func (c *serveCommand) buildApp() (*app.Application, error) {
	application, err := app.New(
		app.WithName(c.appName),
		app.WithLogger(c.log),
		app.WithGracefulTimeout(c.timeout),
	)
	if err != nil {
		return nil, err
	}

	for _, m := range c.modules {
		if regErr := application.Register(m); regErr != nil {
			return nil, regErr
		}
	}

	return application, nil
}
