package command

import (
	"context"
	"fmt"
	"io"

	"github.com/shuldan/cli"
)

type HealthChecker interface {
	Name() string
	Health(ctx context.Context) error
}

func Health(checkers ...HealthChecker) cli.Command {
	return &healthCommand{checkers: checkers}
}

type healthCommand struct {
	checkers []HealthChecker
}

func (c *healthCommand) Name() string          { return "health" }
func (c *healthCommand) Description() string   { return "Check health of all services" }
func (c *healthCommand) Group() string         { return "debug" }
func (c *healthCommand) Args() []cli.Arg       { return nil }
func (c *healthCommand) Options() []cli.Option { return nil }

func (c *healthCommand) Execute(
	ctx context.Context,
	_ io.Reader, out io.Writer, _ *cli.Input,
) error {
	allOK := true

	for _, ch := range c.checkers {
		err := ch.Health(ctx)
		if err != nil {
			_, _ = fmt.Fprintf(out, "  ✗ %s: %v\n", ch.Name(), err)
			allOK = false
		} else {
			_, _ = fmt.Fprintf(out, "  ✓ %s\n", ch.Name())
		}
	}

	if !allOK {
		return fmt.Errorf("health check failed")
	}

	_, _ = fmt.Fprintln(out, "\nAll services healthy.")

	return nil
}
