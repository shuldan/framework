package command

import (
	"context"
	"fmt"
	"io"

	"github.com/shuldan/cli"
	"github.com/shuldan/migrator"

	"github.com/shuldan/framework/migration"
)

func MigrateStatus(runner *migration.Runner) cli.Command {
	return &migrateStatusCommand{runner: runner}
}

type migrateStatusCommand struct {
	runner *migration.Runner
}

func (c *migrateStatusCommand) Name() string        { return "migrate:status" }
func (c *migrateStatusCommand) Description() string { return "Show database migration status" }
func (c *migrateStatusCommand) Group() string       { return databaseGroup }
func (c *migrateStatusCommand) Args() []cli.Arg     { return nil }

func (c *migrateStatusCommand) Options() []cli.Option {
	return []cli.Option{
		cli.StringOption("connection", "c", "",
			"Target database connection (default: all)"),
	}
}

func (c *migrateStatusCommand) Execute(
	ctx context.Context,
	_ io.Reader, out io.Writer, input *cli.Input,
) error {
	conn := input.StringOption("connection")

	results, err := c.runner.Status(ctx, conn)
	if err != nil {
		return err
	}

	writeStatusResults(out, results)

	return nil
}

func writeStatusResults(
	w io.Writer, results []migration.StatusResult,
) {
	for i, res := range results {
		if i > 0 {
			_, _ = fmt.Fprintln(w)
		}

		_, _ = fmt.Fprintf(w, "Connection: %s\n", res.Connection)

		if len(res.Migrations) == 0 {
			_, _ = fmt.Fprintln(w, "  No migrations registered")
			continue
		}

		for _, m := range res.Migrations {
			writeStatusLine(w, m)
		}
	}
}

func writeStatusLine(
	w io.Writer, m migrator.MigrationStatus,
) {
	applied := "-"
	batch := "-"

	if m.AppliedAt != nil {
		applied = m.AppliedAt.Format("2006-01-02 15:04:05")
		batch = fmt.Sprintf("%d", m.Batch)
	}

	_, _ = fmt.Fprintf(w,
		"  [%-7s] %-30s  batch:%-3s  %s\n",
		m.State.String(), m.ID, batch, applied,
	)
}
