package database

import (
	"flag"
	"fmt"
	"io"

	"github.com/shuldan/framework/pkg/contracts"
	"github.com/shuldan/framework/pkg/errors"
)

type migrationStatusCommand struct {
	pool   *databasePool
	config contracts.Config
	logger contracts.Logger
}

func newMigrationStatusCommand(pool *databasePool, config contracts.Config, logger contracts.Logger) contracts.CliCommand {
	return &migrationStatusCommand{pool: pool, config: config, logger: logger}
}

func (c *migrationStatusCommand) Name() string        { return "db:migration:status" }
func (c *migrationStatusCommand) Description() string { return "Show migration status" }
func (c *migrationStatusCommand) Group() string       { return contracts.DatabaseCliGroup }

func (c *migrationStatusCommand) Configure(_ *flag.FlagSet) {}

func (c *migrationStatusCommand) Validate(_ contracts.CliContext) error {
	return nil
}

func (c *migrationStatusCommand) Execute(ctx contracts.CliContext) error {
	out := ctx.Output()
	connectionNames := c.pool.getConnectionNames()
	var errs []error
	for _, connName := range connectionNames {
		_, exists := c.pool.getDatabase(connName)
		if !exists {
			_, _ = fmt.Fprintf(out, "Database connection '%s' not found\n", connName)
			continue
		}
		db, exists := c.pool.getDatabase(connName)
		if !exists {
			continue
		}
		status, err := db.Status()
		if err != nil {
			_, _ = fmt.Fprintf(out, "Failed to get migration status: %s\n", err)
			errs = append(errs, err)
			continue
		}
		_, _ = fmt.Fprintf(out, "\n=== Migration Status for Connection: %s ===\n", connName)
		printTable(out, [][]string{
			{"ID", "Description", "Applied", "Applied At", "Batch"},
		}, func() {
			for _, s := range status {
				applied := "no"
				appliedAt := ""
				if s.AppliedAt != nil {
					applied = "yes"
					appliedAt = s.AppliedAt.Format("2006-01-02 15:04")
				}
				batch := fmt.Sprintf("%d", s.Batch)
				printRow(out, s.ID, s.Description, applied, appliedAt, batch)
			}
		})
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	ctx.AppContext().Stop()
	return nil
}

func printTable(w io.Writer, header [][]string, body func()) {
	for _, row := range header {
		printRow(w, row...)
	}
	body()
}

func printRow(w io.Writer, cols ...string) {
	format := ""
	for range cols {
		format += "%-20s "
	}
	format += "\n"
	_, _ = fmt.Fprintf(w, format, toInterfaceSlice(cols)...)
}

func toInterfaceSlice(slice []string) []interface{} {
	result := make([]interface{}, len(slice))
	for i, v := range slice {
		result[i] = v
	}
	return result
}
