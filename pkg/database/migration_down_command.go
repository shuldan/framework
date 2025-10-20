package database

import (
	"flag"
	"fmt"

	"github.com/shuldan/framework/pkg/contracts"
)

type migrationDownCommand struct {
	migrationAbstractCommand
	config contracts.Config
	logger contracts.Logger
	n      int
}

func newMigrationDownCommand(pool *databasePool, config contracts.Config, logger contracts.Logger) contracts.CliCommand {
	return &migrationDownCommand{
		migrationAbstractCommand: migrationAbstractCommand{
			pool: pool,
		},
		config: config,
		logger: logger,
	}
}

func (c *migrationDownCommand) Name() string { return "db:migration:down" }
func (c *migrationDownCommand) Description() string {
	return "Rollback last N migrations for all connections"
}
func (c *migrationDownCommand) Group() string { return contracts.DatabaseCliGroup }

func (c *migrationDownCommand) Configure(flags *flag.FlagSet) {
	flags.IntVar(&c.n, "n", 1, "Number of migrations to rollback")
	c.migrationAbstractCommand.Configure(flags)
}

func (c *migrationDownCommand) Validate(ctx contracts.CliContext) error {
	return nil
}

func (c *migrationDownCommand) Execute(ctx contracts.CliContext) error {
	if c.n <= 0 {
		c.n = 1
	}

	if err := c.processAllConnections(ctx, func(connName string, db contracts.Database) error {
		_, _ = fmt.Fprintf(ctx.Output(), "Processing migration for connection: %s\n", connName)
		status, err := db.Status()
		if err != nil {
			return err
		}
		applied := 0
		for _, s := range status {
			if s.AppliedAt != nil {
				applied++
			}
		}
		toRollback := c.n
		if c.n > applied {
			toRollback = applied
		}
		if toRollback == 0 {
			_, _ = fmt.Fprintf(ctx.Output(), "No migrations to rollback for connection '%s'\n", connName)
			return nil
		}
		migrations := getMigrations(connName)
		return db.Rollback(toRollback, migrations)
	}); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(ctx.Output(), "All migrations applied successfully")
	ctx.AppContext().Stop()
	return nil
}
