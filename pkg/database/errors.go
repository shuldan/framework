package database

import "github.com/shuldan/framework/pkg/errors"

var newDatabaseCode = errors.WithPrefix("DATABASE")

var (
	ErrEntityNotFound                      = newDatabaseCode().New("entity not found with id {{.id}}")
	ErrInvalidID                           = newDatabaseCode().New("invalid ID: {{.id}}")
	ErrInvalidCriteria                     = newDatabaseCode().New("invalid search criteria: {{.reason}}")
	ErrFailedToOpenDatabase                = newDatabaseCode().New("failed to open database")
	ErrDatabaseNotConnected                = newDatabaseCode().New("database not connected")
	ErrTransactionFailed                   = newDatabaseCode().New("transaction failed: {{.reason}}")
	ErrMigrationFailed                     = newDatabaseCode().New("migration {{.id}} failed: {{.reason}}")
	ErrRepositoryNotInitialized            = newDatabaseCode().New("repository not properly initialized")
	ErrInvalidIDType                       = newDatabaseCode().New("invalid ID type: expected {{.expected}}, got {{.actual}}")
	ErrFailedToCreateSchemaMigrationsTable = newDatabaseCode().New("failed to create schema_migrations table")
	ErrFailedToCreateSchemaMigrationsIndex = newDatabaseCode().New("failed to create schema_migrations index")
	ErrFailedToGetAppliedMigrations        = newDatabaseCode().New("failed to get applied migrations")
	ErrFailedToBeginTransaction            = newDatabaseCode().New("failed to begin transaction")
	ErrNoMigrationsToRollback              = newDatabaseCode().New("no migrations to rollback")
	ErrFailedToExecuteQuery                = newDatabaseCode().New("failed to execute query: {{.query}}")
)
