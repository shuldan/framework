package database

import (
	"flag"
	"fmt"

	"github.com/shuldan/framework/pkg/contracts"
)

type migrationFreshCommand struct {
	migrationAbstractCommand
	config contracts.Config
	logger contracts.Logger
}

func newMigrationFreshCommand(pool *databasePool, config contracts.Config, logger contracts.Logger) contracts.CliCommand {
	return &migrationFreshCommand{
		migrationAbstractCommand: migrationAbstractCommand{
			pool: pool,
		},
		config: config,
		logger: logger,
	}
}

func (c *migrationFreshCommand) Name() string { return "db:migration:fresh" }
func (c *migrationFreshCommand) Description() string {
	return "Drop all tables and re-run all migrations from scratch"
}
func (c *migrationFreshCommand) Group() string { return contracts.DatabaseCliGroup }

func (c *migrationFreshCommand) Configure(flags *flag.FlagSet) {
	c.migrationAbstractCommand.Configure(flags)
}

func (c *migrationFreshCommand) Validate(ctx contracts.CliContext) error {
	if !c.config.GetBool("database.migrations.enabled", true) {
		return ErrMigrationDisabled
	}
	return nil
}

func (c *migrationFreshCommand) Execute(ctx contracts.CliContext) error {
	if err := c.processAllConnections(ctx, func(connName string, db contracts.Database) error {
		_, _ = fmt.Fprintf(ctx.Output(), "Processing fresh migration for connection: %s\n", connName)

		status, err := db.Status()
		if err != nil {
			return err
		}

		if len(status) > 0 {
			_, _ = fmt.Fprintf(ctx.Output(), "Rolling back %d migration(s)...\n", len(status))
			migrations := getMigrations(connName)
			if err := db.Rollback(len(status), migrations); err != nil {
				return err
			}
		}

		_, _ = fmt.Fprintf(ctx.Output(), "Running migrations...\n")
		migrations := getMigrations(connName)

		return db.Migrate(migrations)
	}); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(ctx.Output(), "Fresh migration completed successfully")
	ctx.AppContext().Stop()
	return nil
}
