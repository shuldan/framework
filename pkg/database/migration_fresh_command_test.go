package database

import (
	"bytes"
	"flag"
	"strings"
	"testing"

	"github.com/shuldan/framework/pkg/config"
	"github.com/shuldan/framework/pkg/contracts"
)

func TestMigrationFreshCommand(t *testing.T) {
	testFreshCommandBasics(t)
	testFreshCommandValidation(t)
	testFreshCommandExecution(t)
	testFreshCommandWithNoMigrations(t)
	testFreshCommandWithMultipleConnections(t)
	testFreshCommandConfiguration(t)
}

func testFreshCommandBasics(t *testing.T) {
	t.Run("Basic properties", func(t *testing.T) {
		pool := newDatabasePool()
		cmd := newMigrationFreshCommand(pool, nil, nil)

		if cmd.Name() != "db:migration:fresh" {
			t.Errorf("expected name 'db:migration:fresh', got '%s'", cmd.Name())
		}

		if cmd.Group() != contracts.DatabaseCliGroup {
			t.Errorf("expected group '%s', got '%s'", contracts.DatabaseCliGroup, cmd.Group())
		}

		desc := cmd.Description()
		if !strings.Contains(desc, "Drop all tables") || !strings.Contains(desc, "re-run all migrations") {
			t.Errorf("unexpected description: %s", desc)
		}
	})
}

func testFreshCommandValidation(t *testing.T) {
	t.Run("Validation with migrations disabled", func(t *testing.T) {
		pool := newDatabasePool()
		cfg := config.NewMapConfig(map[string]interface{}{
			"database": map[string]interface{}{
				"migrations": map[string]interface{}{
					"enabled": false,
				},
			},
		})
		cmd := newMigrationFreshCommand(pool, cfg, nil)

		ctx := newMockCliContext()
		err := cmd.Validate(ctx)
		if err == nil {
			t.Error("expected error when migrations disabled")
		}
	})

	t.Run("Validation with migrations enabled", func(t *testing.T) {
		pool := newDatabasePool()
		cfg := config.NewMapConfig(map[string]interface{}{
			"database": map[string]interface{}{
				"migrations": map[string]interface{}{
					"enabled": true,
				},
			},
		})
		cmd := newMigrationFreshCommand(pool, cfg, nil)

		ctx := newMockCliContext()
		err := cmd.Validate(ctx)
		if err != nil {
			t.Errorf("unexpected validation error: %v", err)
		}
	})
}

func testFreshCommandExecution(t *testing.T) {
	t.Run("Execute fresh migration", func(t *testing.T) {
		cleanupMigrationRegistry()
		defer cleanupMigrationRegistry()

		pool := newDatabasePool()
		db := NewDatabase("sqlite3", ":memory:")
		if err := db.Connect(); err != nil {
			t.Fatalf("failed to connect: %v", err)
		}
		defer func(db contracts.Database) {
			_ = db.Close()
		}(db)

		if err := pool.registerDatabase("fresh_test", db); err != nil {
			t.Fatalf("failed to register database: %v", err)
		}

		migration1 := CreateMigration("001", "create users").
			ForConnection("fresh_test").
			CreateTable("users", "id INTEGER PRIMARY KEY", "name TEXT").
			Build()
		migration2 := CreateMigration("002", "create posts").
			ForConnection("fresh_test").
			CreateTable("posts", "id INTEGER PRIMARY KEY", "title TEXT").
			Build()

		registerMigration(migration1)
		registerMigration(migration2)

		registeredMigrations := getMigrations("fresh_test")
		if len(registeredMigrations) != 2 {
			t.Fatalf("expected 2 registered migrations, got %d", len(registeredMigrations))
		}
		err := db.Migrate(registeredMigrations)
		if err != nil {
			t.Fatalf("failed to run initial migrations: %v", err)
		}

		status, err := db.Status()
		if err != nil {
			t.Fatalf("failed to get status: %v", err)
		}
		if len(status) != 2 {
			t.Errorf("expected 2 migrations, got %d", len(status))
		}

		cfg := config.NewMapConfig(map[string]interface{}{})
		cmd := newMigrationFreshCommand(pool, cfg, nil)

		ctx := newMockCliContext()
		err = cmd.Execute(ctx)
		if err != nil {
			t.Errorf("fresh command failed: %v", err)
		}

		output := ctx.Output().(*bytes.Buffer).String()
		if !strings.Contains(output, "Rolling back") {
			t.Error("output should contain 'Rolling back'")
		}
		if !strings.Contains(output, "Running migrations") {
			t.Error("output should contain 'Running migrations'")
		}
		if !strings.Contains(output, "Fresh migration completed successfully") {
			t.Error("output should contain success message")
		}

		status, err = db.Status()
		if err != nil {
			t.Fatalf("failed to get status after fresh: %v", err)
		}

		if len(status) != 2 {
			t.Errorf("expected 2 migrations after fresh, got %d", len(status))
		}

		for _, s := range status {
			if s.Batch != 1 {
				t.Errorf("expected batch 1 after fresh, got %d", s.Batch)
			}
		}

		if ctx.Ctx().IsRunning() {
			t.Error("app context should be stopped after execution")
		}
	})
}

func testFreshCommandWithNoMigrations(t *testing.T) {
	t.Run("Execute fresh with no migrations", func(t *testing.T) {
		cleanupMigrationRegistry()
		defer cleanupMigrationRegistry()

		pool := newDatabasePool()
		db := NewDatabase("sqlite3", ":memory:")
		if err := db.Connect(); err != nil {
			t.Fatalf("failed to connect: %v", err)
		}
		defer func(db contracts.Database) {
			_ = db.Close()
		}(db)

		if err := pool.registerDatabase("empty", db); err != nil {
			t.Fatalf("failed to register database: %v", err)
		}

		cfg := config.NewMapConfig(map[string]interface{}{})
		cmd := newMigrationFreshCommand(pool, cfg, nil)

		ctx := newMockCliContext()
		err := cmd.Execute(ctx)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		output := ctx.Output().(*bytes.Buffer).String()
		if !strings.Contains(output, "Fresh migration completed successfully") {
			t.Error("output should contain success message")
		}
	})
}

func testFreshCommandWithMultipleConnections(t *testing.T) {
	t.Run("Execute fresh with multiple connections", func(t *testing.T) {
		cleanupMigrationRegistry()
		defer cleanupMigrationRegistry()

		pool := newDatabasePool()
		defer func(pool *databasePool) {
			_ = pool.closeAll()
		}(pool)

		for i := 0; i < 2; i++ {
			db := NewDatabase("sqlite3", ":memory:")
			if err := db.Connect(); err != nil {
				t.Fatalf("failed to connect db%d: %v", i, err)
			}

			name := string(rune('a' + i))
			if err := pool.registerDatabase(name, db); err != nil {
				t.Fatalf("failed to register database %s: %v", name, err)
			}

			migration := CreateMigration("001", "create table").
				ForConnection(name).
				CreateTable("test_table", "id INTEGER PRIMARY KEY").
				Build()
			registerMigration(migration)

			if err := db.Migrate([]contracts.Migration{migration}); err != nil {
				t.Fatalf("failed to run migration for %s: %v", name, err)
			}
		}

		cfg := config.NewMapConfig(map[string]interface{}{})
		cmd := newMigrationFreshCommand(pool, cfg, nil)

		ctx := newMockCliContext()
		err := cmd.Execute(ctx)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		output := ctx.Output().(*bytes.Buffer).String()
		if !strings.Contains(output, "connection: a") {
			t.Error("output should contain connection 'a'")
		}
		if !strings.Contains(output, "connection: b") {
			t.Error("output should contain connection 'b'")
		}
	})
}

func testFreshCommandConfiguration(t *testing.T) {
	t.Run("Configure flags", func(t *testing.T) {
		pool := newDatabasePool()
		cmd := newMigrationFreshCommand(pool, nil, nil)

		flags := flag.NewFlagSet("test", flag.ContinueOnError)
		cmd.Configure(flags)

		connFlag := flags.Lookup("connection")
		if connFlag == nil {
			t.Error("expected 'connection' flag to be configured")
		}

		cFlag := flags.Lookup("c")
		if cFlag == nil {
			t.Error("expected 'c' shorthand flag to be configured")
		}
	})
}

func TestFreshCommandWithSpecificConnection(t *testing.T) {
	t.Run("Execute fresh for specific connection", func(t *testing.T) {
		cleanupMigrationRegistry()
		defer cleanupMigrationRegistry()

		pool := newDatabasePool()
		defer func(pool *databasePool) {
			_ = pool.closeAll()
		}(pool)

		db1 := NewDatabase("sqlite3", ":memory:")
		if err := db1.Connect(); err != nil {
			t.Fatalf("failed to connect db1: %v", err)
		}
		if err := pool.registerDatabase("conn1", db1); err != nil {
			t.Fatalf("failed to register db1: %v", err)
		}

		db2 := NewDatabase("sqlite3", ":memory:")
		if err := db2.Connect(); err != nil {
			t.Fatalf("failed to connect db2: %v", err)
		}
		if err := pool.registerDatabase("conn2", db2); err != nil {
			t.Fatalf("failed to register db2: %v", err)
		}

		migration1 := CreateMigration("001", "create table").
			ForConnection("conn1").
			CreateTable("test_table1", "id INTEGER PRIMARY KEY").
			Build()
		migration2 := CreateMigration("002", "create table").
			ForConnection("conn2").
			CreateTable("test_table2", "id INTEGER PRIMARY KEY").
			Build()

		registerMigration(migration1)
		registerMigration(migration2)

		if err := db1.Migrate([]contracts.Migration{migration1}); err != nil {
			t.Fatalf("failed to run migration for conn1: %v", err)
		}
		if err := db2.Migrate([]contracts.Migration{migration2}); err != nil {
			t.Fatalf("failed to run migration for conn2: %v", err)
		}

		cfg := config.NewMapConfig(map[string]interface{}{})
		cmd := newMigrationFreshCommand(pool, cfg, nil)

		flags := flag.NewFlagSet("test", flag.ContinueOnError)
		cmd.Configure(flags)
		_ = flags.Set("connection", "conn1")

		ctx := newMockCliContext()
		err := cmd.Execute(ctx)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		output := ctx.Output().(*bytes.Buffer).String()

		if !strings.Contains(output, "connection: conn1") {
			t.Error("output should contain only conn1")
		}
		if strings.Contains(output, "connection: conn2") {
			t.Error("output should not contain conn2")
		}

		status1, _ := db1.Status()
		status2, _ := db2.Status()

		if len(status1) != 1 {
			t.Errorf("expected 1 migration for conn1, got %d", len(status1))
		}
		if len(status2) != 1 {
			t.Errorf("expected 1 migration for conn2 (unchanged), got %d", len(status2))
		}
	})
}

func TestFreshCommandIntegration(t *testing.T) {
	t.Run("Full fresh workflow", func(t *testing.T) {
		cleanupMigrationRegistry()
		defer cleanupMigrationRegistry()

		db, pool, cfg, lg := setupIntegrationTestEnv(t)
		defer func(db contracts.Database) {
			_ = db.Close()
		}(db)

		upCmd := newMigrationUpCommand(pool, cfg, lg)
		ctx := newMockCliContext()
		if err := upCmd.Execute(ctx); err != nil {
			t.Fatalf("up command failed: %v", err)
		}

		status, _ := db.Status()
		if len(status) != 3 {
			t.Errorf("expected 3 migrations before fresh, got %d", len(status))
		}

		freshCmd := newMigrationFreshCommand(pool, cfg, lg)
		ctx = newMockCliContext()
		if err := freshCmd.Execute(ctx); err != nil {
			t.Fatalf("fresh command failed: %v", err)
		}

		status, _ = db.Status()
		if len(status) != 3 {
			t.Errorf("expected 3 migrations after fresh, got %d", len(status))
		}

		for _, s := range status {
			if s.Batch != 1 {
				t.Errorf("expected batch 1 after fresh, got %d for migration %s", s.Batch, s.ID)
			}
		}
	})
}
