package database

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/shuldan/framework/pkg/app"
	"github.com/shuldan/framework/pkg/config"
	"github.com/shuldan/framework/pkg/contracts"
	"github.com/shuldan/framework/pkg/errors"
)

type mockCliContext struct {
	appCtx contracts.AppContext
	input  io.Reader
	output io.Writer
	args   []string
}

func newMockCliContext() *mockCliContext {
	return &mockCliContext{
		appCtx: newMockAppContext(),
		input:  strings.NewReader(""),
		output: &bytes.Buffer{},
		args:   []string{},
	}
}

func (m *mockCliContext) Ctx() contracts.AppContext {
	return m.appCtx
}

func (m *mockCliContext) Input() io.Reader {
	return m.input
}

func (m *mockCliContext) Output() io.Writer {
	return m.output
}

func (m *mockCliContext) Args() []string {
	return m.args
}

type mockAppContext struct {
	ctx         context.Context
	container   contracts.DIContainer
	appName     string
	version     string
	environment string
	startTime   time.Time
	stopTime    time.Time
	running     bool
	registry    contracts.AppRegistry
}

func newMockAppContext() *mockAppContext {
	return &mockAppContext{
		ctx:         context.Background(),
		container:   app.NewContainer(),
		appName:     "test-app",
		version:     "1.0.0",
		environment: "test",
		startTime:   time.Now(),
		running:     true,
		registry:    app.NewRegistry(),
	}
}

func (m *mockAppContext) Ctx() context.Context {
	return m.ctx
}

func (m *mockAppContext) Container() contracts.DIContainer {
	return m.container
}

func (m *mockAppContext) AppName() string {
	return m.appName
}

func (m *mockAppContext) Version() string {
	return m.version
}

func (m *mockAppContext) Environment() string {
	return m.environment
}

func (m *mockAppContext) StartTime() time.Time {
	return m.startTime
}

func (m *mockAppContext) StopTime() time.Time {
	return m.stopTime
}

func (m *mockAppContext) IsRunning() bool {
	return m.running
}

func (m *mockAppContext) Stop() {
	m.running = false
	m.stopTime = time.Now()
}

func (m *mockAppContext) AppRegistry() contracts.AppRegistry {
	return m.registry
}

type mockLogger struct {
	messages []string
	mu       sync.Mutex
}

func (m *mockLogger) log(level, msg string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	formatted := fmt.Sprintf("[%s] %s", level, msg)
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			formatted += fmt.Sprintf(" %v=%v", args[i], args[i+1])
		}
	}
	m.messages = append(m.messages, formatted)
}

func (m *mockLogger) Trace(msg string, args ...any) { m.log("TRACE", msg, args...) }
func (m *mockLogger) Debug(msg string, args ...any) { m.log("DEBUG", msg, args...) }
func (m *mockLogger) Info(msg string, args ...any)  { m.log("INFO", msg, args...) }
func (m *mockLogger) Warn(msg string, args ...any)  { m.log("WARN", msg, args...) }
func (m *mockLogger) Error(msg string, args ...any) { m.log("ERROR", msg, args...) }
func (m *mockLogger) Critical(msg string, args ...any) {
	m.log("CRITICAL", msg, args...)
}
func (m *mockLogger) With(_ ...any) contracts.Logger { return m }

func TestMigrationUpCommand(t *testing.T) {
	t.Parallel()

	testUpCommandName(t)
	testUpCommandDescription(t)
	testUpCommandGroup(t)
	testUpCommandValidation(t)
	testUpCommandExecute(t)
	testUpCommandWithSpecificConnection(t)
	testUpCommandWithMultipleConnections(t)
	testUpCommandWithMigrationErrors(t)
}

func testUpCommandName(t *testing.T) {
	t.Run("Name", func(t *testing.T) {
		pool := newDatabasePool()
		cmd := newMigrationUpCommand(pool, nil, nil)
		if cmd.Name() != "db:migration:up" {
			t.Errorf("expected name 'db:migration:up', got '%s'", cmd.Name())
		}
	})
}

func testUpCommandDescription(t *testing.T) {
	t.Run("Description", func(t *testing.T) {
		pool := newDatabasePool()
		cmd := newMigrationUpCommand(pool, nil, nil)
		desc := cmd.Description()
		if !strings.Contains(desc, "Migrate all pending migrations") {
			t.Errorf("unexpected description: %s", desc)
		}
	})
}

func testUpCommandGroup(t *testing.T) {
	t.Run("Group", func(t *testing.T) {
		pool := newDatabasePool()
		cmd := newMigrationUpCommand(pool, nil, nil)
		if cmd.Group() != contracts.DatabaseCliGroup {
			t.Errorf("expected group '%s', got '%s'", contracts.DatabaseCliGroup, cmd.Group())
		}
	})
}

func testUpCommandValidation(t *testing.T) {
	t.Run("Validation with migrations disabled", func(t *testing.T) {
		pool := newDatabasePool()
		cfg := config.NewMapConfig(map[string]interface{}{
			"database": map[string]interface{}{
				"migrations": map[string]interface{}{
					"enabled": false,
				},
			},
		})
		cmd := newMigrationUpCommand(pool, cfg, nil)

		ctx := newMockCliContext()
		err := cmd.Validate(ctx)
		if !errors.Is(err, ErrMigrationDisabled) {
			t.Errorf("expected ErrMigrationDisabled, got %v", err)
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
		cmd := newMigrationUpCommand(pool, cfg, nil)

		ctx := newMockCliContext()
		err := cmd.Validate(ctx)
		if err != nil {
			t.Errorf("unexpected validation error: %v", err)
		}
	})
}

func testUpCommandExecute(t *testing.T) {
	t.Run("Execute with no connections", func(t *testing.T) {
		pool := newDatabasePool()
		cfg := config.NewMapConfig(map[string]interface{}{})
		cmd := newMigrationUpCommand(pool, cfg, nil)

		ctx := newMockCliContext()

		err := cmd.Execute(ctx)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		output := ctx.Output().(*bytes.Buffer).String()
		if !strings.Contains(output, "All migrations applied successfully") {
			t.Errorf("unexpected output: %s", output)
		}

		if ctx.Ctx().IsRunning() {
			t.Error("app context should be stopped after execution")
		}
	})
}

func testUpCommandWithSpecificConnection(t *testing.T) {
	t.Run("Execute with specific connection", func(t *testing.T) {
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

		if err := pool.registerDatabase("test", db); err != nil {
			t.Fatalf("failed to register database: %v", err)
		}

		migration := CreateMigration("001", "test migration").
			ForConnection("test").
			CreateTable("test_table", "id INTEGER PRIMARY KEY").
			Build()
		registerMigration(migration)

		cfg := config.NewMapConfig(map[string]interface{}{})
		cmd := newMigrationUpCommand(pool, cfg, nil)

		flags := flag.NewFlagSet("test", flag.ContinueOnError)
		cmd.Configure(flags)
		_ = flags.Set("connection", "test")

		ctx := newMockCliContext()

		err := cmd.Execute(ctx)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		output := ctx.Output().(*bytes.Buffer).String()
		if !strings.Contains(output, "Processing migration for connection: test") {
			t.Errorf("unexpected output: %s", output)
		}
	})
}

func testUpCommandWithMultipleConnections(t *testing.T) {
	t.Run("Execute with multiple connections", func(t *testing.T) {
		cleanupMigrationRegistry()
		defer cleanupMigrationRegistry()

		pool := newDatabasePool()
		defer func(pool *databasePool) {
			_ = pool.closeAll()
		}(pool)

		for i := 0; i < 3; i++ {
			db := NewDatabase("sqlite3", ":memory:")
			if err := db.Connect(); err != nil {
				t.Fatalf("failed to connect db%d: %v", i, err)
			}

			name := fmt.Sprintf("conn%d", i)
			if err := pool.registerDatabase(name, db); err != nil {
				t.Fatalf("failed to register database %s: %v", name, err)
			}
			migration := CreateMigration(fmt.Sprintf("01%d", i+1), fmt.Sprintf("migration %d", i)).
				ForConnection(name).
				CreateTable(fmt.Sprintf("table_%d", i), "id INTEGER PRIMARY KEY").
				Build()
			registerMigration(migration)
		}

		cfg := config.NewMapConfig(map[string]interface{}{})
		cmd := newMigrationUpCommand(pool, cfg, nil)
		ctx := newMockCliContext()
		err := cmd.Execute(ctx)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		output := ctx.Output().(*bytes.Buffer).String()
		for i := 0; i < 3; i++ {
			expected := fmt.Sprintf("Processing migration for connection: conn%d", i)
			if !strings.Contains(output, expected) {
				t.Errorf("output should contain '%s'", expected)
			}
		}
	})
}

func testUpCommandWithMigrationErrors(t *testing.T) {
	t.Run("Execute with migration errors", func(t *testing.T) {
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

		if err := pool.registerDatabase("error_test", db); err != nil {
			t.Fatalf("failed to register database: %v", err)
		}

		migration := CreateMigration("001", "bad migration").
			ForConnection("error_test").
			RawUp("INVALID SQL SYNTAX").
			Build()
		registerMigration(migration)

		cfg := config.NewMapConfig(map[string]interface{}{})
		cmd := newMigrationUpCommand(pool, cfg, nil)

		ctx := newMockCliContext()

		err := cmd.Execute(ctx)
		if err == nil {
			t.Error("expected error for invalid migration")
		}
	})
}

func TestMigrationDownCommand(t *testing.T) {
	t.Parallel()

	testDownCommandBasics(t)
	testDownCommandConfiguration(t)
	testDownCommandExecution(t)
	testDownCommandWithNoMigrations(t)
	testDownCommandWithMultipleSteps(t)
}

func testDownCommandBasics(t *testing.T) {
	t.Run("Basic properties", func(t *testing.T) {
		pool := newDatabasePool()
		cmd := newMigrationDownCommand(pool, nil, nil)

		if cmd.Name() != "db:migration:down" {
			t.Errorf("expected name 'db:migration:down', got '%s'", cmd.Name())
		}

		if cmd.Group() != contracts.DatabaseCliGroup {
			t.Errorf("expected group '%s', got '%s'", contracts.DatabaseCliGroup, cmd.Group())
		}

		desc := cmd.Description()
		if !strings.Contains(desc, "Rollback") {
			t.Errorf("unexpected description: %s", desc)
		}
	})
}

func testDownCommandConfiguration(t *testing.T) {
	t.Run("Configure flags", func(t *testing.T) {
		pool := newDatabasePool()
		cmd := newMigrationDownCommand(pool, nil, nil)

		flags := flag.NewFlagSet("test", flag.ContinueOnError)
		cmd.Configure(flags)

		nFlag := flags.Lookup("n")
		if nFlag == nil {
			t.Error("expected 'n' flag to be configured")
		}

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

func testDownCommandExecution(t *testing.T) {
	t.Run("Execute rollback", func(t *testing.T) {
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

		if err := pool.registerDatabase("rollback_test", db); err != nil {
			t.Fatalf("failed to register database: %v", err)
		}

		migration1 := CreateMigration("001", "first").
			ForConnection("rollback_test").
			CreateTable("users", "id INTEGER PRIMARY KEY").
			Build()
		migration2 := CreateMigration("002", "second").
			ForConnection("rollback_test").
			CreateTable("posts", "id INTEGER PRIMARY KEY").
			Build()

		registerMigration(migration1)
		registerMigration(migration2)

		err := db.Migrate([]contracts.Migration{migration1, migration2})
		if err != nil {
			t.Fatalf("failed to run migrations: %v", err)
		}

		cfg := config.NewMapConfig(map[string]interface{}{})
		downCmd := &migrationDownCommand{
			migrationAbstractCommand: migrationAbstractCommand{pool: pool},
			config:                   cfg,
			n:                        1,
		}

		ctx := newMockCliContext()

		err = downCmd.Execute(ctx)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		status, _ := db.Status()
		if len(status) != 1 {
			t.Errorf("expected 1 migration after rollback, got %d", len(status))
		}
	})
}

func testDownCommandWithNoMigrations(t *testing.T) {
	t.Run("Execute with no migrations to rollback", func(t *testing.T) {
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
		cmd := newMigrationDownCommand(pool, cfg, nil)

		ctx := newMockCliContext()

		err := cmd.Execute(ctx)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		output := ctx.Output().(*bytes.Buffer).String()
		if !strings.Contains(output, "No migrations to rollback") {
			t.Errorf("expected 'No migrations to rollback' in output")
		}
	})
}

func testDownCommandWithMultipleSteps(t *testing.T) {
	t.Run("Execute with multiple steps", func(t *testing.T) {
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

		if err := pool.registerDatabase("multi", db); err != nil {
			t.Fatalf("failed to register database: %v", err)
		}

		migrations := make([]contracts.Migration, 5)
		for i := 0; i < 5; i++ {
			migrations[i] = CreateMigration(fmt.Sprintf("00%d", i+1), fmt.Sprintf("migration %d", i)).
				ForConnection("multi").
				CreateTable(fmt.Sprintf("table_%d", i), "id INTEGER PRIMARY KEY").
				Build()
			registerMigration(migrations[i])
		}

		err := db.Migrate(migrations)
		if err != nil {
			t.Fatalf("failed to run migrations: %v", err)
		}

		cfg := config.NewMapConfig(map[string]interface{}{})
		downCmd := &migrationDownCommand{
			migrationAbstractCommand: migrationAbstractCommand{pool: pool},
			config:                   cfg,
			n:                        3,
		}

		ctx := newMockCliContext()

		err = downCmd.Execute(ctx)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		status, _ := db.Status()
		if len(status) != 2 {
			t.Errorf("expected 2 migrations remaining, got %d", len(status))
		}
	})
}

func TestMigrationStatusCommand(t *testing.T) {
	t.Parallel()

	testStatusCommandBasics(t)
	testStatusCommandExecute(t)
	testStatusCommandWithMultipleConnections(t)
	testStatusCommandWithErrors(t)
}

func testStatusCommandBasics(t *testing.T) {
	t.Run("Basic properties", func(t *testing.T) {
		pool := newDatabasePool()
		cmd := newMigrationStatusCommand(pool, nil, nil)

		if cmd.Name() != "db:migration:status" {
			t.Errorf("expected name 'db:migration:status', got '%s'", cmd.Name())
		}

		if cmd.Group() != contracts.DatabaseCliGroup {
			t.Errorf("expected group '%s', got '%s'", contracts.DatabaseCliGroup, cmd.Group())
		}

		desc := cmd.Description()
		if !strings.Contains(desc, "Show migration status") {
			t.Errorf("unexpected description: %s", desc)
		}

		flags := flag.NewFlagSet("test", flag.ContinueOnError)
		cmd.Configure(flags)
	})
}

func testStatusCommandExecute(t *testing.T) {
	t.Run("Execute with migrations", func(t *testing.T) {
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

		if err := pool.registerDatabase("status_test", db); err != nil {
			t.Fatalf("failed to register database: %v", err)
		}

		migration := CreateMigration("001", "test migration").
			ForConnection("status_test").
			CreateTable("test", "id INTEGER PRIMARY KEY").
			Build()
		registerMigration(migration)

		err := db.Migrate([]contracts.Migration{migration})
		if err != nil {
			t.Fatalf("failed to run migration: %v", err)
		}

		cfg := config.NewMapConfig(map[string]interface{}{})
		cmd := newMigrationStatusCommand(pool, cfg, nil)

		ctx := newMockCliContext()

		err = cmd.Execute(ctx)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		output := ctx.Output().(*bytes.Buffer).String()
		if !strings.Contains(output, "001") {
			t.Error("output should contain migration ID")
		}
		if !strings.Contains(output, "test migration") {
			t.Error("output should contain migration description")
		}
		if !strings.Contains(output, "yes") {
			t.Error("output should show migration as applied")
		}
	})
}

func testStatusCommandWithMultipleConnections(t *testing.T) {
	t.Run("Execute with multiple connections", func(t *testing.T) {
		cleanupMigrationRegistry()
		defer cleanupMigrationRegistry()

		pool := newDatabasePool()
		defer func(pool *databasePool) {
			_ = pool.closeAll()
		}(pool)

		for i := 0; i < 2; i++ {
			db := NewDatabase("sqlite3", ":memory:")
			if err := db.Connect(); err != nil {
				t.Fatalf("failed to connect: %v", err)
			}

			name := fmt.Sprintf("conn%d", i)
			if err := pool.registerDatabase(name, db); err != nil {
				t.Fatalf("failed to register database: %v", err)
			}

			migration := CreateMigration(fmt.Sprintf("00%d", i+1), fmt.Sprintf("migration %d", i)).
				ForConnection(name).
				CreateTable(fmt.Sprintf("table_%d", i), "id INTEGER PRIMARY KEY").
				Build()
			registerMigration(migration)

			if i == 0 {
				err := db.Migrate([]contracts.Migration{migration})
				if err != nil {
					t.Fatalf("failed to run migration: %v", err)
				}
			}
		}

		cfg := config.NewMapConfig(map[string]interface{}{})
		cmd := newMigrationStatusCommand(pool, cfg, nil)

		ctx := newMockCliContext()

		err := cmd.Execute(ctx)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		output := ctx.Output().(*bytes.Buffer).String()
		if !strings.Contains(output, "conn0") {
			t.Error("output should contain first connection")
		}
		if !strings.Contains(output, "conn1") {
			t.Error("output should contain second connection")
		}
	})
}

func testStatusCommandWithErrors(t *testing.T) {
	t.Run("Execute with no connections", func(t *testing.T) {
		pool := newDatabasePool()
		cfg := config.NewMapConfig(map[string]interface{}{})
		cmd := newMigrationStatusCommand(pool, cfg, nil)

		ctx := newMockCliContext()

		err := cmd.Execute(ctx)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if ctx.Ctx().IsRunning() {
			t.Error("app context should be stopped")
		}
	})
}

func TestMigrationAbstractCommand(t *testing.T) {
	testMigrationAbstractCommandProcessAllConnections(t)
	testMigrationAbstractCommandProcessAllConnectionWithError(t)
	testMigrationAbstractCommandConfigureFlags(t)
}

func testMigrationAbstractCommandProcessAllConnections(t *testing.T) {
	t.Run("processAllConnections", func(t *testing.T) {
		cleanupMigrationRegistry()
		defer cleanupMigrationRegistry()

		pool := newDatabasePool()

		db1 := NewDatabase("sqlite3", ":memory:")
		_ = db1.Connect()
		defer func(db contracts.Database) {
			_ = db.Close()
		}(db1)
		_ = pool.registerDatabase("conn1", db1)

		db2 := NewDatabase("sqlite3", ":memory:")
		_ = db2.Connect()
		defer func(db contracts.Database) {
			_ = db.Close()
		}(db2)
		_ = pool.registerDatabase("conn2", db2)

		registerMigration(CreateMigration("001", "test").ForConnection("conn1").Build())
		registerMigration(CreateMigration("002", "test").ForConnection("conn2").Build())

		cmd := &migrationAbstractCommand{pool: pool}

		processedConns := make(map[string]bool)
		err := cmd.processAllConnections(nil, func(connName string, db contracts.Database) error {
			processedConns[connName] = true
			return nil
		})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(processedConns) != 2 {
			t.Errorf("expected 2 connections processed, got %d", len(processedConns))
		}
	})
}

func testMigrationAbstractCommandProcessAllConnectionWithError(t *testing.T) {
	t.Run("processAllConnections with errors", func(t *testing.T) {
		cleanupMigrationRegistry()
		defer cleanupMigrationRegistry()

		pool := newDatabasePool()

		db := NewDatabase("sqlite3", ":memory:")
		_ = db.Connect()
		defer func(db contracts.Database) {
			_ = db.Close()
		}(db)
		_ = pool.registerDatabase("error_conn", db)

		registerMigration(CreateMigration("001", "test").ForConnection("error_conn").Build())

		cmd := &migrationAbstractCommand{pool: pool}

		err := cmd.processAllConnections(nil, func(connName string, db contracts.Database) error {
			return fmt.Errorf("test error")
		})

		if err == nil {
			t.Error("expected error")
		}
		if !strings.Contains(err.Error(), "test error") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func testMigrationAbstractCommandConfigureFlags(t *testing.T) {
	t.Run("Configure flags", func(t *testing.T) {
		cmd := &migrationAbstractCommand{}
		flags := flag.NewFlagSet("test", flag.ContinueOnError)

		cmd.Configure(flags)

		if flags.Lookup("connection") == nil {
			t.Error("expected 'connection' flag")
		}
		if flags.Lookup("c") == nil {
			t.Error("expected 'c' shorthand flag")
		}
	})
}

func TestCliCommandsIntegration(t *testing.T) {
	t.Run("Full migration workflow", func(t *testing.T) {
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

		statusCmd := newMigrationStatusCommand(pool, cfg, lg)
		ctx = newMockCliContext()
		if err := statusCmd.Execute(ctx); err != nil {
			t.Fatalf("status command failed: %v", err)
		}

		status, _ := db.Status()
		if len(status) != 3 {
			t.Errorf("expected 3 migrations applied, got %d", len(status))
		}

		downCmd := &migrationDownCommand{
			migrationAbstractCommand: migrationAbstractCommand{pool: pool},
			config:                   cfg,
			logger:                   lg,
			n:                        1,
		}
		ctx = newMockCliContext()
		if err := downCmd.Execute(ctx); err != nil {
			t.Fatalf("down command failed: %v", err)
		}

		status, _ = db.Status()
		if len(status) != 2 {
			t.Errorf("expected 2 migrations after rollback, got %d", len(status))
		}
	})
}

func setupIntegrationTestEnv(t *testing.T) (contracts.Database, *databasePool, contracts.Config, contracts.Logger) {
	db := NewDatabase("sqlite3", ":memory:")
	if err := db.Connect(); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	pool := newDatabasePool()
	if err := pool.registerDatabase("primary", db); err != nil {
		t.Fatalf("failed to register database: %v", err)
	}

	migrations := []contracts.Migration{
		CreateMigration("001", "create users").
			ForConnection("primary").
			CreateTable("users", "id INTEGER PRIMARY KEY", "name TEXT").
			Build(),
		CreateMigration("002", "create posts").
			ForConnection("primary").
			CreateTable("posts", "id INTEGER PRIMARY KEY", "title TEXT").
			Build(),
		CreateMigration("003", "add index").
			ForConnection("primary").
			CreateIndex("idx_posts_title", "posts", "title").
			Build(),
	}

	for _, m := range migrations {
		registerMigration(m)
	}

	cfg := config.NewMapConfig(map[string]interface{}{})
	logger := &mockLogger{
		messages: make([]string, 0),
	}

	return db, pool, cfg, logger
}
