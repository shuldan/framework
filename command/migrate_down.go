package command

import (
	"context"
	"fmt"
	"io"

	"github.com/shuldan/cli"

	"github.com/shuldan/framework/migration"
)

func MigrateDown(runner *migration.Runner) cli.Command {
	return &migrateDownCommand{runner: runner}
}

type migrateDownCommand struct {
	runner *migration.Runner
}

func (c *migrateDownCommand) Name() string        { return "migrate:down" }
func (c *migrateDownCommand) Description() string { return "Rollback database migrations" }
func (c *migrateDownCommand) Group() string       { return databaseGroup }
func (c *migrateDownCommand) Args() []cli.Arg     { return nil }

func (c *migrateDownCommand) Options() []cli.Option {
	return []cli.Option{
		cli.StringOption("connection", "c", "",
			"Target database connection (default: all)"),
		cli.IntOption("steps", "s", 1,
			"Number of migrations to rollback"),
		cli.BoolOption("force", "f", false,
			"Force rollback of irreversible migrations"),
	}
}

func (c *migrateDownCommand) Execute(
	ctx context.Context,
	_ io.Reader, out io.Writer, input *cli.Input,
) error {
	conn := input.StringOption("connection")
	steps := input.IntOption("steps")
	force := input.BoolOption("force")

	if err := c.runner.Down(ctx, conn, steps, force); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(out, "Migrations rolled back successfully.")

	return nil
}
