package migration

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/shuldan/migrator"

	"github.com/shuldan/framework/database"
)

func init() {
	sql.Register("migtestdb", &migTestDBDriver{})
}

func TestNewRunner(t *testing.T) {
	t.Parallel()
	r := NewRunner(nil, nil)
	if r == nil {
		t.Fatal("expected non-nil Runner")
	}
}

func TestRunner_Register(t *testing.T) {
	t.Parallel()
	r := NewRunner(nil, nil)
	r.Register("default", buildMigration("001", "first"), buildMigration("002", "second"))
	r.Register("analytics", buildMigration("003", "third"))
	names := r.ConnectionNames()
	if len(names) != 2 {
		t.Fatalf("expected 2 connections, got %d", len(names))
	}
	if names[0] != "analytics" || names[1] != "default" {
		t.Fatalf("expected [analytics,default], got %v", names)
	}
}

func TestRunner_Register_Appends(t *testing.T) {
	t.Parallel()
	r := NewRunner(nil, nil)
	r.Register("default", buildMigration("001", "first"))
	r.Register("default", buildMigration("002", "second"))
	if len(r.ConnectionNames()) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(r.ConnectionNames()))
	}
}

func TestRunner_ConnectionNames_Empty(t *testing.T) {
	t.Parallel()
	r := NewRunner(nil, nil)
	if len(r.ConnectionNames()) != 0 {
		t.Fatalf("expected 0 connections, got %d", len(r.ConnectionNames()))
	}
}

func TestRunnerOption_WithMigrationTable(t *testing.T) {
	t.Parallel()
	r := NewRunner(nil, nil, WithMigrationTable("custom_migrations"))
	if r.tableName != "custom_migrations" {
		t.Fatalf("expected 'custom_migrations', got %q", r.tableName)
	}
}

func TestRunnerOption_WithAdvisoryLock(t *testing.T) {
	t.Parallel()
	r := NewRunner(nil, nil, WithAdvisoryLock())
	if !r.lock {
		t.Fatal("expected lock to be true")
	}
}

func TestDriverToDialect(t *testing.T) {
	t.Parallel()
	tests := []struct {
		driver   string
		expected migrator.Dialect
	}{
		{"postgres", migrator.DialectPostgreSQL},
		{"pgx", migrator.DialectPostgreSQL},
		{"POSTGRES", migrator.DialectPostgreSQL},
		{"mysql", migrator.DialectMySQL},
		{"MYSQL", migrator.DialectMySQL},
		{"sqlite3", migrator.DialectSQLite},
		{"sqlite", migrator.DialectSQLite},
		{"unknown", migrator.DialectUnknown},
		{"", migrator.DialectUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.driver, func(t *testing.T) {
			t.Parallel()
			got := driverToDialect(tt.driver)
			if got != tt.expected {
				t.Errorf("driverToDialect(%q) = %v, want %v", tt.driver, got, tt.expected)
			}
		})
	}
}

func TestEnsureLog_Nil(t *testing.T) {
	t.Parallel()
	l := ensureLog(nil)
	if _, ok := l.(noopLogger); !ok {
		t.Fatal("expected noopLogger")
	}
}

func TestEnsureLog_NonNil(t *testing.T) {
	t.Parallel()
	ml := &migMockLogger{}
	if ensureLog(ml) != ml {
		t.Fatal("expected same logger")
	}
}

func TestRunner_Targets_Specific(t *testing.T) {
	t.Parallel()
	r := NewRunner(nil, nil)
	r.Register("default", buildMigration("001", "first"))
	r.Register("analytics", buildMigration("002", "second"))
	targets := r.targets("default")
	if len(targets) != 1 || targets[0] != "default" {
		t.Fatalf("expected [default], got %v", targets)
	}
}

func TestRunner_Targets_All(t *testing.T) {
	t.Parallel()
	r := NewRunner(nil, nil)
	r.Register("default", buildMigration("001", "first"))
	r.Register("analytics", buildMigration("002", "second"))
	if len(r.targets("")) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(r.targets("")))
	}
}

func TestRunner_Up_ConnectionNotFound(t *testing.T) {
	t.Parallel()
	dbm := newMigTestDBManager(t)
	defer func() { _ = dbm.Stop(context.Background()) }()
	r := NewRunner(dbm, nil)
	r.Register("missing", buildMigration("001", "first"))
	err := r.Up(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, database.ErrConnectionNotFound) {
		t.Fatalf("expected ErrConnectionNotFound, got %v", err)
	}
}

func TestRunner_Up_SQLError(t *testing.T) {
	t.Parallel()
	dbm := newMigTestDBManager(t)
	defer func() { _ = dbm.Stop(context.Background()) }()
	r := NewRunner(dbm, nil)
	r.Register("default", buildMigration("001", "first"))
	err := r.Up(context.Background(), "default")
	if err == nil {
		t.Fatal("expected SQL error from migrator Up")
	}
}

func TestRunner_Down_ConnectionNotFound(t *testing.T) {
	t.Parallel()
	dbm := newMigTestDBManager(t)
	defer func() { _ = dbm.Stop(context.Background()) }()
	r := NewRunner(dbm, nil)
	r.Register("missing", buildMigration("001", "first"))
	err := r.Down(context.Background(), "missing", 1, false)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunner_Down_SQLError_NoForce(t *testing.T) {
	t.Parallel()
	dbm := newMigTestDBManager(t)
	defer func() { _ = dbm.Stop(context.Background()) }()
	r := NewRunner(dbm, nil)
	r.Register("default", buildMigration("001", "first"))
	err := r.Down(context.Background(), "default", 1, false)
	if err == nil {
		t.Fatal("expected SQL error from migrator Down")
	}
}

func TestRunner_Down_SQLError_WithForce(t *testing.T) {
	t.Parallel()
	dbm := newMigTestDBManager(t)
	defer func() { _ = dbm.Stop(context.Background()) }()
	r := NewRunner(dbm, nil)
	r.Register("default", buildMigration("001", "first"))
	err := r.Down(context.Background(), "default", 1, true)
	if err == nil {
		t.Fatal("expected SQL error from migrator Down")
	}
}

func TestRunner_Status_ConnectionNotFound(t *testing.T) {
	t.Parallel()
	dbm := newMigTestDBManager(t)
	defer func() { _ = dbm.Stop(context.Background()) }()
	r := NewRunner(dbm, nil)
	r.Register("missing", buildMigration("001", "first"))
	_, err := r.Status(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunner_Status_SQLError(t *testing.T) {
	t.Parallel()
	dbm := newMigTestDBManager(t)
	defer func() { _ = dbm.Stop(context.Background()) }()
	r := NewRunner(dbm, nil)
	r.Register("default", buildMigration("001", "first"))
	_, err := r.Status(context.Background(), "default")
	if err == nil {
		t.Fatal("expected SQL error from migrator Status")
	}
}

func TestRunner_Plan_ConnectionNotFound(t *testing.T) {
	t.Parallel()
	dbm := newMigTestDBManager(t)
	defer func() { _ = dbm.Stop(context.Background()) }()
	r := NewRunner(dbm, nil)
	r.Register("missing", buildMigration("001", "first"))
	_, err := r.Plan(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunner_Plan_SQLError(t *testing.T) {
	t.Parallel()
	dbm := newMigTestDBManager(t)
	defer func() { _ = dbm.Stop(context.Background()) }()
	r := NewRunner(dbm, nil)
	r.Register("default", buildMigration("001", "first"))
	_, err := r.Plan(context.Background(), "default")
	if err == nil {
		t.Fatal("expected SQL error from migrator Plan")
	}
}

func TestRunner_BuildMigrator_Success(t *testing.T) {
	t.Parallel()
	dbm := newMigTestDBManager(t)
	defer func() { _ = dbm.Stop(context.Background()) }()
	r := NewRunner(dbm, nil, WithMigrationTable("test_mig"), WithAdvisoryLock())
	r.Register("default", buildMigration("001", "first"))
	m, err := r.buildMigrator("default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil migrator")
	}
}

func TestRunner_BuildMigrator_NoMigrationsForConn(t *testing.T) {
	t.Parallel()
	dbm := newMigTestDBManager(t)
	defer func() { _ = dbm.Stop(context.Background()) }()
	r := NewRunner(dbm, nil)
	m, err := r.buildMigrator("default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil migrator")
	}
}

func TestRunner_ForEachTarget_Error(t *testing.T) {
	t.Parallel()
	r := NewRunner(nil, nil)
	r.Register("a", buildMigration("001", "first"))
	err := r.forEachTarget("a", func(_ string) error {
		return errors.New("fail")
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunner_ForEachTarget_All(t *testing.T) {
	t.Parallel()
	dbm := newMigTestDBManager(t)
	defer func() { _ = dbm.Stop(context.Background()) }()
	r := NewRunner(dbm, nil)
	r.Register("default", buildMigration("001", "first"))
	var called []string
	err := r.forEachTarget("", func(name string) error {
		called = append(called, name)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(called) != 1 || called[0] != "default" {
		t.Fatalf("expected [default], got %v", called)
	}
}

func TestRunner_MigratorOpts(t *testing.T) {
	t.Parallel()
	dbm := newMigTestDBManager(t)
	defer func() { _ = dbm.Stop(context.Background()) }()
	r := NewRunner(dbm, nil, WithMigrationTable("mig"), WithAdvisoryLock())
	opts := r.migratorOpts("default")
	if len(opts) < 4 {
		t.Fatalf("expected at least 4 opts, got %d", len(opts))
	}
}

func TestNoopMigLogger(t *testing.T) {
	t.Parallel()
	l := noopLogger{}
	l.Info("test")
	l.Error("test")
}

func buildMigration(id, desc string) migrator.Migration {
	return migrator.CreateMigration(id, desc).
		RawUp("SELECT 1").
		MustBuild()
}

func newMigTestDBManager(t *testing.T) *database.Manager {
	t.Helper()
	configs := map[string]database.ConnectionConfig{
		"default": {Driver: "migtestdb", DSN: "test"},
	}
	mgr, err := database.NewManager(configs, nil)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	return mgr
}

type migMockLogger struct{}

func (m *migMockLogger) Info(_ string, _ ...any)  {}
func (m *migMockLogger) Error(_ string, _ ...any) {}

type migTestDBDriver struct{}

func (d *migTestDBDriver) Open(_ string) (driver.Conn, error) {
	return &migTestDBConn{}, nil
}

type migTestDBConn struct{}

func (c *migTestDBConn) Prepare(_ string) (driver.Stmt, error) {
	return nil, driver.ErrSkip
}

func (c *migTestDBConn) Close() error              { return nil }
func (c *migTestDBConn) Begin() (driver.Tx, error) { return nil, driver.ErrSkip }
