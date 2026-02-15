package command

import (
	"context"
	"fmt"
	"io"

	"github.com/shuldan/cli"

	"github.com/shuldan/framework/migration"
)

func MigrateUp(runner *migration.Runner) cli.Command {
	return &migrateUpCommand{runner: runner}
}

type migrateUpCommand struct {
	runner *migration.Runner
}

func (c *migrateUpCommand) Name() string        { return "migrate:up" }
func (c *migrateUpCommand) Description() string { return "Apply pending database migrations" }
func (c *migrateUpCommand) Group() string       { return databaseGroup }
func (c *migrateUpCommand) Args() []cli.Arg     { return nil }

func (c *migrateUpCommand) Options() []cli.Option {
	return []cli.Option{
		cli.StringOption("connection", "c", "",
			"Target database connection (default: all)"),
	}
}

func (c *migrateUpCommand) Execute(
	ctx context.Context,
	_ io.Reader, out io.Writer, input *cli.Input,
) error {
	conn := input.StringOption("connection")

	if err := c.runner.Up(ctx, conn); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(out, "Migrations applied successfully.")

	return nil
}
