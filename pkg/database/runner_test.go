package database

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/shuldan/framework/pkg/contracts"
	"testing"
)

func TestMigrationRunner(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	runner := NewMigrationRunner(db)

	t.Run("CreateMigrationTable", func(t *testing.T) {
		err := runner.CreateMigrationTable()
		if err != nil {
			t.Errorf("failed to create migration table: %v", err)
		}

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'").Scan(&count)
		if err != nil {
			t.Errorf("failed to check table existence: %v", err)
		}
		if count != 1 {
			t.Error("schema_migrations table was not created")
		}

		err = runner.CreateMigrationTable()
		if err != nil {
			t.Errorf("second CreateMigrationTable call failed: %v", err)
		}
	})

	t.Run("Run empty migrations", func(t *testing.T) {
		err := runner.Run([]contracts.Migration{})
		if err != nil {
			t.Errorf("running empty migrations failed: %v", err)
		}
	})

	t.Run("Run single migration", func(t *testing.T) {
		migration := CreateMigration("001", "create users table").
			CreateTable("users", "id INTEGER PRIMARY KEY", "name TEXT NOT NULL").
			Build()

		err := runner.Run([]contracts.Migration{migration})
		if err != nil {
			t.Errorf("failed to run migration: %v", err)
		}

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='users'").Scan(&count)
		if err != nil {
			t.Errorf("failed to check users table: %v", err)
		}
		if count != 1 {
			t.Error("users table was not created")
		}

		var migrationCount int
		err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE id = '001'").Scan(&migrationCount)
		if err != nil {
			t.Errorf("failed to check migration record: %v", err)
		}
		if migrationCount != 1 {
			t.Error("migration was not recorded")
		}
	})

	t.Run("Run multiple migrations", func(t *testing.T) {
		migration2 := CreateMigration("002", "create posts table").
			CreateTable("posts", "id INTEGER PRIMARY KEY", "title TEXT NOT NULL", "user_id INTEGER").
			Build()

		migration3 := CreateMigration("003", "add index").
			CreateIndex("idx_posts_user_id", "posts", "user_id").
			Build()

		err := runner.Run([]contracts.Migration{migration2, migration3})
		if err != nil {
			t.Errorf("failed to run migrations: %v", err)
		}

		tables := []string{"posts"}
		for _, table := range tables {
			var count int
			err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
			if err != nil {
				t.Errorf("failed to check %s table: %v", table, err)
			}
			if count != 1 {
				t.Errorf("%s table was not created", table)
			}
		}

		var batch1, batch2 int
		err = db.QueryRow("SELECT batch FROM schema_migrations WHERE id = '002'").Scan(&batch1)
		if err != nil {
			t.Errorf("failed to get batch for migration 002: %v", err)
		}
		err = db.QueryRow("SELECT batch FROM schema_migrations WHERE id = '003'").Scan(&batch2)
		if err != nil {
			t.Errorf("failed to get batch for migration 003: %v", err)
		}

		if batch1 != batch2 {
			t.Error("migrations in same run should have same batch number")
		}
		if batch1 <= 1 {
			t.Error("new batch should be greater than previous batch")
		}
	})

	t.Run("Skip already applied migrations", func(t *testing.T) {

		migration1 := CreateMigration("001", "create users table").
			CreateTable("users", "id INTEGER PRIMARY KEY", "name TEXT NOT NULL").
			Build()

		err := runner.Run([]contracts.Migration{migration1})
		if err != nil {
			t.Errorf("failed to run already applied migration: %v", err)
		}

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE id = '001'").Scan(&count)
		if err != nil {
			t.Errorf("failed to check migration count: %v", err)
		}
		if count != 1 {
			t.Error("migration should not be applied twice")
		}
	})

	t.Run("Status", func(t *testing.T) {
		status, err := runner.Status()
		if err != nil {
			t.Errorf("failed to get migration status: %v", err)
		}

		if len(status) < 3 {
			t.Errorf("expected at least 3 migrations, got %d", len(status))
		}

		firstMigration := status[0]
		if firstMigration.ID != "001" {
			t.Errorf("expected first migration ID '001', got '%s'", firstMigration.ID)
		}
		if firstMigration.Description != "create users table" {
			t.Errorf("unexpected description: %s", firstMigration.Description)
		}
		if firstMigration.AppliedAt == nil {
			t.Error("AppliedAt should not be nil")
		}
		if firstMigration.Batch != 1 {
			t.Errorf("expected batch 1, got %d", firstMigration.Batch)
		}
	})

	t.Run("Rollback", func(t *testing.T) {

		migrations := []contracts.Migration{
			CreateMigration("001", "create users table").
				CreateTable("users", "id INTEGER PRIMARY KEY", "name TEXT NOT NULL").
				Build(),
			CreateMigration("002", "create posts table").
				CreateTable("posts", "id INTEGER PRIMARY KEY", "title TEXT NOT NULL", "user_id INTEGER").
				Build(),
			CreateMigration("003", "add index").
				CreateIndex("idx_posts_user_id", "posts", "user_id").
				Build(),
		}

		err := runner.Rollback(1, migrations)
		if err != nil {
			t.Errorf("failed to rollback: %v", err)
		}

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE id = '003'").Scan(&count)
		if err != nil {
			t.Errorf("failed to check migration record: %v", err)
		}
		if count != 0 {
			t.Error("migration 003 should have been rolled back")
		}

		err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_posts_user_id'").Scan(&count)
		if err != nil {
			t.Errorf("failed to check index: %v", err)
		}
		if count != 0 {
			t.Error("index should have been dropped")
		}

		err = runner.Rollback(2, migrations)
		if err != nil {
			t.Errorf("failed to rollback multiple steps: %v", err)
		}

		err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
		if err != nil {
			t.Errorf("failed to check migration count: %v", err)
		}
		if count != 0 {
			t.Error("all migrations should have been rolled back")
		}

		tables := []string{"users", "posts"}
		for _, table := range tables {
			err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
			if err != nil {
				t.Errorf("failed to check %s table: %v", table, err)
			}
			if count != 0 {
				t.Errorf("%s table should have been dropped", table)
			}
		}
	})

	t.Run("Rollback with no migrations", func(t *testing.T) {
		err := runner.Rollback(1, []contracts.Migration{})
		if err != ErrNoMigrationsToRollback {
			t.Errorf("expected ErrNoMigrationsToRollback, got %v", err)
		}
	})

	t.Run("Failed migration rollback", func(t *testing.T) {

		migration := CreateMigration("004", "test table").
			CreateTable("test_table", "id INTEGER PRIMARY KEY").
			Build()

		err := runner.Run([]contracts.Migration{migration})
		if err != nil {
			t.Errorf("failed to run migration: %v", err)
		}

		badMigration := CreateMigration("004", "test table").
			RawUp("CREATE TABLE test_table (id INTEGER PRIMARY KEY);").
			RawDown("INVALID SQL QUERY;").
			Build()

		err = runner.Rollback(1, []contracts.Migration{badMigration})
		if err == nil {
			t.Error("expected rollback to fail with invalid SQL")
		}
	})
}

func TestMigrationRunnerTransactions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	runner := NewMigrationRunner(db)

	t.Run("Transaction rollback on failure", func(t *testing.T) {

		migration1 := CreateMigration("001", "create users").
			CreateTable("users", "id INTEGER PRIMARY KEY").
			Build()

		migration2 := CreateMigration("002", "invalid migration").
			RawUp("INVALID SQL SYNTAX;").
			Build()

		err := runner.Run([]contracts.Migration{migration1, migration2})
		if err == nil {
			t.Error("expected migration to fail")
		}

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE name='schema_migrations'").Scan(&count)
		if err != nil {
			t.Error("schema_migrations table should not exist after rollback")
		}

		err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='users'").Scan(&count)
		if err == nil && count > 0 {
			t.Error("users table should not exist after rollback")
		}
	})
}

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	return db
}
