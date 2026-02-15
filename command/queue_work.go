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

func QueueWork(
	name string,
	log *logger.Logger,
	timeout time.Duration,
	modules ...app.Module,
) cli.Command {
	return &queueWorkCommand{
		appName: name,
		log:     log,
		timeout: timeout,
		modules: modules,
	}
}

type queueWorkCommand struct {
	appName string
	log     *logger.Logger
	timeout time.Duration
	modules []app.Module
}

func (c *queueWorkCommand) Name() string          { return "queue:work" }
func (c *queueWorkCommand) Description() string   { return "Start queue consumer workers" }
func (c *queueWorkCommand) Group() string         { return "queue" }
func (c *queueWorkCommand) Args() []cli.Arg       { return nil }
func (c *queueWorkCommand) Options() []cli.Option { return nil }

func (c *queueWorkCommand) Execute(
	ctx context.Context, _ io.Reader, _ io.Writer, _ *cli.Input,
) error {
	application, err := c.buildApp()
	if err != nil {
		return fmt.Errorf("queue:work: build app: %w", err)
	}

	return application.Run(ctx)
}

func (c *queueWorkCommand) buildApp() (*app.Application, error) {
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
