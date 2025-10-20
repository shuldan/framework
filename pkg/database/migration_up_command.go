package database

import (
	"fmt"

	"github.com/shuldan/framework/pkg/contracts"
)

type migrationUpCommand struct {
	migrationAbstractCommand
	config contracts.Config
	logger contracts.Logger
}

func newMigrationUpCommand(pool *databasePool, config contracts.Config, logger contracts.Logger) contracts.CliCommand {
	return &migrationUpCommand{
		migrationAbstractCommand: migrationAbstractCommand{
			pool: pool,
		},
		config: config,
		logger: logger,
	}
}

func (c *migrationUpCommand) Name() string { return "db:migration:up" }
func (c *migrationUpCommand) Description() string {
	return "Migrate all pending migrations from modules and global paths"
}
func (c *migrationUpCommand) Group() string { return contracts.DatabaseCliGroup }

func (c *migrationUpCommand) Validate(ctx contracts.CliContext) error {
	if !c.config.GetBool("database.migrations.enabled", true) {
		return ErrMigrationDisabled
	}
	return nil
}

func (c *migrationUpCommand) Execute(ctx contracts.CliContext) error {
	if err := c.processAllConnections(ctx, func(connName string, db contracts.Database) error {
		_, _ = fmt.Fprintf(ctx.Output(), "Processing migration for connection: %s\n", connName)
		migrations := getMigrations(connName)
		return db.Migrate(migrations)
	}); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(ctx.Output(), "All migrations applied successfully")
	ctx.AppContext().Stop()
	return nil
}
