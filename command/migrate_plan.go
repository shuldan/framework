package command

import (
	"context"
	"fmt"
	"io"

	"github.com/shuldan/cli"
	"github.com/shuldan/migrator"

	"github.com/shuldan/framework/migration"
)

func MigratePlan(runner *migration.Runner) cli.Command {
	return &migratePlanCommand{runner: runner}
}

type migratePlanCommand struct {
	runner *migration.Runner
}

func (c *migratePlanCommand) Name() string        { return "migrate:plan" }
func (c *migratePlanCommand) Description() string { return "Show pending migration SQL" }
func (c *migratePlanCommand) Group() string       { return databaseGroup }
func (c *migratePlanCommand) Args() []cli.Arg     { return nil }

func (c *migratePlanCommand) Options() []cli.Option {
	return []cli.Option{
		cli.StringOption("connection", "c", "",
			"Target database connection (default: all)"),
	}
}

func (c *migratePlanCommand) Execute(
	ctx context.Context,
	_ io.Reader, out io.Writer, input *cli.Input,
) error {
	conn := input.StringOption("connection")

	results, err := c.runner.Plan(ctx, conn)
	if err != nil {
		return err
	}

	writePlanResults(out, results)

	return nil
}

func writePlanResults(
	w io.Writer, results []migration.PlanResult,
) {
	for i, res := range results {
		if i > 0 {
			_, _ = fmt.Fprintln(w)
		}

		_, _ = fmt.Fprintf(w, "Connection: %s\n", res.Connection)

		if len(res.Migrations) == 0 {
			_, _ = fmt.Fprintln(w, "  No pending migrations")
			continue
		}

		for _, m := range res.Migrations {
			writePlanMigration(w, m)
		}
	}
}

func writePlanMigration(
	w io.Writer, m migrator.PlannedMigration,
) {
	_, _ = fmt.Fprintf(w,
		"\n  %s — %s\n", m.ID, m.Description,
	)

	for _, q := range m.Queries {
		_, _ = fmt.Fprintf(w, "    %s\n", q)
	}
}
