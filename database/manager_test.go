package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"
	"time"
)

func init() {
	sql.Register("testdb", &testDBDriver{})
	sql.Register("failpingdb", &failPingDriver{})
}

func TestNewManager_Success(t *testing.T) {
	t.Parallel()
	configs := map[string]ConnectionConfig{
		"default": {Driver: "testdb", DSN: "test"},
		"other":   {Driver: "testdb", DSN: "test2"},
	}
	mgr, err := NewManager(configs, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = mgr.Stop(context.Background()) }()
	if mgr.Name() != "database" {
		t.Errorf("expected 'database', got %q", mgr.Name())
	}
}

func TestNewManager_NoConfigs(t *testing.T) {
	t.Parallel()
	_, err := NewManager(nil, nil)
	if !errors.Is(err, ErrNoConnections) {
		t.Fatalf("expected ErrNoConnections, got %v", err)
	}
}

func TestNewManager_EmptyConfigs(t *testing.T) {
	t.Parallel()
	_, err := NewManager(map[string]ConnectionConfig{}, nil)
	if !errors.Is(err, ErrNoConnections) {
		t.Fatalf("expected ErrNoConnections, got %v", err)
	}
}

func TestNewManager_InvalidDriver(t *testing.T) {
	t.Parallel()
	configs := map[string]ConnectionConfig{
		"default": {Driver: "nonexistent", DSN: "fake"},
	}
	_, err := NewManager(configs, nil)
	if err == nil {
		t.Fatal("expected error for invalid driver")
	}
}

func TestNewManager_WithLogger(t *testing.T) {
	t.Parallel()
	log := &mockLogger{}
	configs := map[string]ConnectionConfig{
		"default": {Driver: "testdb", DSN: "test"},
	}
	mgr, err := NewManager(configs, log)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = mgr.Stop(context.Background()) }()
	if !log.infoCalled {
		t.Error("expected logger Info to be called")
	}
}

func TestManager_Connection(t *testing.T) {
	t.Parallel()
	mgr := newTestManager(t)
	defer stopManager(t, mgr)
	if mgr.Connection("default") == nil {
		t.Fatal("expected non-nil *sql.DB")
	}
}

func TestManager_Default(t *testing.T) {
	t.Parallel()
	mgr := newTestManager(t)
	defer stopManager(t, mgr)
	if mgr.Default() == nil {
		t.Fatal("expected non-nil *sql.DB from Default()")
	}
}

func TestManager_Connection_Panics(t *testing.T) {
	t.Parallel()
	mgr := newTestManager(t)
	defer stopManager(t, mgr)
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for missing connection")
		}
	}()
	mgr.Connection("nonexistent")
}

func TestManager_Driver(t *testing.T) {
	t.Parallel()
	mgr := newTestManager(t)
	defer stopManager(t, mgr)
	if d := mgr.Driver("default"); d != "testdb" {
		t.Errorf("expected 'testdb', got %q", d)
	}
	if d := mgr.Driver("missing"); d != "" {
		t.Errorf("expected empty, got %q", d)
	}
}

func TestManager_Names(t *testing.T) {
	t.Parallel()
	configs := map[string]ConnectionConfig{
		"beta": {Driver: "testdb", DSN: "t"}, "alpha": {Driver: "testdb", DSN: "t"},
		"default": {Driver: "testdb", DSN: "t"},
	}
	mgr, err := NewManager(configs, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer stopManager(t, mgr)
	names := mgr.Names()
	expected := []string{"alpha", "beta", "default"}
	if len(names) != len(expected) {
		t.Fatalf("expected %d names, got %d", len(expected), len(names))
	}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("index %d: expected %q, got %q", i, expected[i], name)
		}
	}
}

func TestManager_Has(t *testing.T) {
	t.Parallel()
	mgr := newTestManager(t)
	defer stopManager(t, mgr)
	if !mgr.Has("default") {
		t.Error("expected Has('default') = true")
	}
	if mgr.Has("missing") {
		t.Error("expected Has('missing') = false")
	}
}

func TestManager_Init_IsNoop(t *testing.T) {
	t.Parallel()
	mgr := newTestManager(t)
	defer stopManager(t, mgr)
	if err := mgr.Init(context.Background()); err != nil {
		t.Fatalf("Init should be no-op, got: %v", err)
	}
}

func TestManager_Start_PingsAll(t *testing.T) {
	t.Parallel()
	mgr := newTestManager(t)
	defer stopManager(t, mgr)
	if err := mgr.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
}

func TestManager_Health(t *testing.T) {
	t.Parallel()
	mgr := newTestManager(t)
	defer stopManager(t, mgr)
	if err := mgr.Health(context.Background()); err != nil {
		t.Fatalf("Health: %v", err)
	}
}

func TestManager_PingAll_Error(t *testing.T) {
	t.Parallel()
	configs := map[string]ConnectionConfig{
		"default": {Driver: "failpingdb", DSN: "test"},
	}
	mgr, err := NewManager(configs, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = mgr.Stop(context.Background()) }()
	err = mgr.Start(context.Background())
	if err == nil {
		t.Fatal("expected ping error from failpingdb")
	}
	err = mgr.Health(context.Background())
	if err == nil {
		t.Fatal("expected health error from failpingdb")
	}
}

func TestManager_PoolConfig(t *testing.T) {
	t.Parallel()
	configs := map[string]ConnectionConfig{
		"default": {
			Driver: "testdb", DSN: "test",
			MaxOpenConns: 10, MaxIdleConns: 5, ConnMaxLifetime: 30 * time.Second,
		},
	}
	mgr, err := NewManager(configs, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer stopManager(t, mgr)
	if mgr.Connection("default") == nil {
		t.Fatal("expected non-nil *sql.DB")
	}
}

func TestManager_PoolConfig_ZeroValues(t *testing.T) {
	t.Parallel()
	configs := map[string]ConnectionConfig{
		"default": {Driver: "testdb", DSN: "test"},
	}
	mgr, err := NewManager(configs, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stopManager(t, mgr)
}

func TestNoopLogger(t *testing.T) {
	t.Parallel()
	l := noopLogger{}
	l.Info("test")
	l.Error("test")
}

func TestEnsureLogger_Nil(t *testing.T) {
	t.Parallel()
	l := ensureLogger(nil)
	if _, ok := l.(noopLogger); !ok {
		t.Fatal("expected noopLogger")
	}
}

func TestEnsureLogger_NonNil(t *testing.T) {
	t.Parallel()
	ml := &mockLogger{}
	if ensureLogger(ml) != ml {
		t.Fatal("expected same logger")
	}
}

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	configs := map[string]ConnectionConfig{
		"default": {Driver: "testdb", DSN: "test"},
	}
	mgr, err := NewManager(configs, nil)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	return mgr
}

func stopManager(t *testing.T, mgr *Manager) {
	t.Helper()
	if err := mgr.Stop(context.Background()); err != nil {
		t.Errorf("stop error: %v", err)
	}
}

type mockLogger struct {
	infoCalled  bool
	errorCalled bool
}

func (m *mockLogger) Info(_ string, _ ...any)  { m.infoCalled = true }
func (m *mockLogger) Error(_ string, _ ...any) { m.errorCalled = true }

type testDBDriver struct{}

func (d *testDBDriver) Open(_ string) (driver.Conn, error) {
	return &testDBConn{}, nil
}

type testDBConn struct{}

func (c *testDBConn) Prepare(_ string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (c *testDBConn) Close() error                          { return nil }
func (c *testDBConn) Begin() (driver.Tx, error)             { return nil, driver.ErrSkip }
func (c *testDBConn) Ping(_ context.Context) error          { return nil }
func (c *testDBConn) IsValid() bool                         { return true }
func (c *testDBConn) ResetSession(_ context.Context) error  { return nil }

type failPingDriver struct{}

func (d *failPingDriver) Open(_ string) (driver.Conn, error) {
	return &failPingConn{}, nil
}

type failPingConn struct{}

func (c *failPingConn) Prepare(_ string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (c *failPingConn) Close() error                          { return nil }
func (c *failPingConn) Begin() (driver.Tx, error)             { return nil, driver.ErrSkip }
func (c *failPingConn) Ping(_ context.Context) error          { return errors.New("ping failed") }
