package database

import (
	"context"
	"database/sql"
	"sort"
	"strings"
	"time"

	"github.com/shuldan/framework/pkg/contracts"
)

type sqlMigrationRunner struct {
	db *sql.DB
}

func NewMigrationRunner(db *sql.DB) contracts.MigrationRunner {
	return &sqlMigrationRunner{db: db}
}

const migrationTableSQL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    id VARCHAR(255) PRIMARY KEY,
    description TEXT NOT NULL,
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    batch INTEGER NOT NULL
);
`

const migrationTableIndexSQL = `
CREATE INDEX IF NOT EXISTS idx_schema_migrations_batch ON schema_migrations(batch);
`

func (r *sqlMigrationRunner) CreateMigrationTable() error {
	_, err := r.db.Exec(migrationTableSQL)
	if err != nil {
		return ErrFailedToCreateSchemaMigrationsTable.WithCause(err)
	}

	_, err = r.db.Exec(migrationTableIndexSQL)
	if err != nil {
		return ErrFailedToCreateSchemaMigrationsIndex.WithCause(err)
	}

	return nil
}

func (r *sqlMigrationRunner) Run(migrations []contracts.Migration) error {
	ctx := context.Background()

	if err := r.CreateMigrationTable(); err != nil {
		return err
	}

	applied, err := r.getAppliedMigrations(ctx)
	if err != nil {
		return ErrFailedToGetAppliedMigrations.WithCause(err)
	}

	appliedMap := make(map[string]bool)
	for _, a := range applied {
		appliedMap[a.ID] = true
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].ID() < migrations[j].ID()
	})

	nextBatch := r.getNextBatchNumber(applied)

	var newMigrations []contracts.Migration
	for _, migration := range migrations {
		if !appliedMap[migration.ID()] {
			newMigrations = append(newMigrations, migration)
		}
	}

	if len(newMigrations) == 0 {
		return nil
	}

	return r.executeMigrationBatch(ctx, newMigrations, nextBatch)
}

func (r *sqlMigrationRunner) executeMigrationBatch(ctx context.Context, migrations []contracts.Migration, batch int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ErrFailedToBeginTransaction.WithCause(err)
	}

	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	for _, migration := range migrations {
		if err := r.executeMigrationUp(ctx, tx, migration, batch); err != nil {
			return ErrMigrationFailed.
				WithDetail("id", migration.ID()).
				WithDetail("reason", err.Error()).
				WithCause(err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	tx = nil
	return nil
}

func (r *sqlMigrationRunner) Rollback(steps int, migrations []contracts.Migration) error {
	ctx := context.Background()

	applied, err := r.getAppliedMigrations(ctx)
	if err != nil {
		return ErrFailedToGetAppliedMigrations.WithCause(err)
	}

	if len(applied) == 0 {
		return ErrNoMigrationsToRollback
	}

	migrationMap := r.buildMigrationMap(migrations)
	rollbackList := r.buildRollbackList(applied, steps)

	return r.executeRollback(ctx, rollbackList, migrationMap)
}

func (r *sqlMigrationRunner) buildMigrationMap(migrations []contracts.Migration) map[string]contracts.Migration {
	migrationMap := make(map[string]contracts.Migration)
	for _, m := range migrations {
		migrationMap[m.ID()] = m
	}
	return migrationMap
}

func (r *sqlMigrationRunner) buildRollbackList(applied []contracts.MigrationStatus, steps int) []contracts.MigrationStatus {
	sort.Slice(applied, func(i, j int) bool {
		return applied[i].Batch > applied[j].Batch ||
			(applied[i].Batch == applied[j].Batch && applied[i].ID > applied[j].ID)
	})

	if steps <= 0 || steps > len(applied) {
		steps = len(applied)
	}

	return applied[:steps]
}

func (r *sqlMigrationRunner) executeRollback(ctx context.Context, rollbackList []contracts.MigrationStatus, migrationMap map[string]contracts.Migration) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ErrFailedToBeginTransaction.WithCause(err)
	}

	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	for _, migrationStatus := range rollbackList {
		if err := r.rollbackSingleMigration(ctx, tx, migrationStatus, migrationMap); err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	tx = nil
	return nil
}

func (r *sqlMigrationRunner) rollbackSingleMigration(ctx context.Context, tx *sql.Tx, migrationStatus contracts.MigrationStatus, migrationMap map[string]contracts.Migration) error {
	if migration, exists := migrationMap[migrationStatus.ID]; exists {
		for _, query := range migration.Down() {
			trimmedQuery := strings.TrimSpace(query)
			if trimmedQuery == "" || strings.HasPrefix(trimmedQuery, "--") {
				continue
			}

			if _, err := tx.ExecContext(ctx, query); err != nil {
				return ErrMigrationFailed.
					WithDetail("id", migrationStatus.ID).
					WithDetail("reason", "rollback query failed: "+err.Error()).
					WithCause(err)
			}
		}
	}

	if err := r.deleteMigrationRecord(ctx, tx, migrationStatus.ID); err != nil {
		return ErrMigrationFailed.
			WithDetail("id", migrationStatus.ID).
			WithDetail("reason", "failed to delete migration record").
			WithCause(err)
	}

	return nil
}

func (r *sqlMigrationRunner) Status() ([]contracts.MigrationStatus, error) {
	return r.getAppliedMigrations(context.Background())
}

func (r *sqlMigrationRunner) executeMigrationUp(ctx context.Context, tx *sql.Tx, migration contracts.Migration, batch int) error {
	for _, query := range migration.Up() {
		if strings.TrimSpace(query) == "" {
			continue
		}

		if _, err := tx.ExecContext(ctx, query); err != nil {
			return ErrFailedToExecuteQuery.
				WithDetail("query", query).
				WithCause(err)
		}
	}

	_, err := tx.ExecContext(ctx,
		"INSERT INTO schema_migrations (id, description, batch) VALUES (?, ?, ?)",
		migration.ID(), migration.Description(), batch)

	return err
}

func (r *sqlMigrationRunner) deleteMigrationRecord(ctx context.Context, tx *sql.Tx, migrationID string) error {
	_, err := tx.ExecContext(ctx, "DELETE FROM schema_migrations WHERE id = ?", migrationID)
	return err
}

func (r *sqlMigrationRunner) getAppliedMigrations(ctx context.Context) ([]contracts.MigrationStatus, error) {
	query := "SELECT id, description, applied_at, batch FROM schema_migrations ORDER BY batch, id"
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}

	defer func() {
		if rows != nil {
			_ = rows.Close()
		}
	}()

	var migrations []contracts.MigrationStatus
	for rows.Next() {
		var migration contracts.MigrationStatus
		var appliedAt time.Time

		err := rows.Scan(&migration.ID, &migration.Description, &appliedAt, &migration.Batch)
		if err != nil {
			return nil, err
		}

		migration.AppliedAt = &appliedAt
		migrations = append(migrations, migration)
	}

	return migrations, rows.Err()
}

func (r *sqlMigrationRunner) getNextBatchNumber(applied []contracts.MigrationStatus) int {
	maxBatch := 0
	for _, migration := range applied {
		if migration.Batch > maxBatch {
			maxBatch = migration.Batch
		}
	}
	return maxBatch + 1
}
