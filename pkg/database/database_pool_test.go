package database

import (
	"context"
	"sync"
	"testing"

	"github.com/shuldan/framework/pkg/contracts"
	"github.com/shuldan/framework/pkg/errors"
)

func TestNewDatabasePool(t *testing.T) {
	t.Parallel()

	pool := newDatabasePool()
	if pool == nil {
		t.Fatal("newDatabasePool returned nil")
	}

	if pool.connections == nil {
		t.Error("connections map should be initialized")
	}

	if len(pool.connections) != 0 {
		t.Error("connections map should be empty initially")
	}
}

func TestRegisterDatabase(t *testing.T) {
	t.Parallel()
	t.Run("successful registration", func(t *testing.T) {
		t.Parallel()
		pool := newDatabasePool()
		db := NewDatabase("sqlite3", ":memory:")
		err := pool.registerDatabase("test", db)
		if err != nil {
			t.Errorf("failed to register database: %v", err)
		}
		if len(pool.connections) != 1 {
			t.Errorf("expected 1 connection, got %d", len(pool.connections))
		}
	})
	t.Run("duplicate registration", func(t *testing.T) {
		t.Parallel()
		pool := newDatabasePool()
		db1 := NewDatabase("sqlite3", ":memory:")
		db2 := NewDatabase("sqlite3", ":memory:")
		err := pool.registerDatabase("test", db1)
		if err != nil {
			t.Fatalf("failed to register first database: %v", err)
		}
		err = pool.registerDatabase("test", db2)
		if err == nil {
			t.Error("expected error for duplicate registration")
		}
		if !errors.Is(err, ErrRegisterConnection) {
			t.Errorf("expected ErrRegisterConnection, got %v", err)
		}
	})
	t.Run("concurrent registration", func(t *testing.T) {
		t.Parallel()
		pool := newDatabasePool()
		var wg sync.WaitGroup
		errs := make(chan error, 10)
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				db := NewDatabase("sqlite3", ":memory:")
				err := pool.registerDatabase("conn"+string(rune(id)), db)
				if err != nil {
					errs <- err
				}
			}(i)
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			t.Errorf("concurrent registration failed: %v", err)
		}
		if len(pool.connections) != 10 {
			t.Errorf("expected 10 connections, got %d", len(pool.connections))
		}
	})
}

func TestGetDatabase(t *testing.T) {
	t.Parallel()

	t.Run("existing database", func(t *testing.T) {
		t.Parallel()
		pool := newDatabasePool()
		db := NewDatabase("sqlite3", ":memory:")
		_ = pool.registerDatabase("test", db)

		gotDB, exists := pool.getDatabase("test")
		if !exists {
			t.Error("database should exist")
		}
		if gotDB != db {
			t.Error("returned database should be the same instance")
		}
	})

	t.Run("non-existing database", func(t *testing.T) {
		t.Parallel()
		pool := newDatabasePool()

		_, exists := pool.getDatabase("non-existent")
		if exists {
			t.Error("database should not exist")
		}
	})

	t.Run("concurrent reads", func(t *testing.T) {
		t.Parallel()
		pool := newDatabasePool()
		db := NewDatabase("sqlite3", ":memory:")
		_ = pool.registerDatabase("test", db)

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				gotDB, exists := pool.getDatabase("test")
				if !exists {
					t.Error("database should exist")
				}
				if gotDB != db {
					t.Error("returned database should be the same instance")
				}
			}()
		}
		wg.Wait()
	})
}

func TestConnectAll(t *testing.T) {
	t.Parallel()

	t.Run("all successful connections", func(t *testing.T) {
		t.Parallel()
		pool := newDatabasePool()

		for i := 0; i < 3; i++ {
			db := NewDatabase("sqlite3", ":memory:")
			_ = pool.registerDatabase(string(rune('a'+i)), db)
		}

		err := pool.connectAll()
		if err != nil {
			t.Errorf("connectAll failed: %v", err)
		}

		for name := range pool.connections {
			db, _ := pool.getDatabase(name)
			if err := db.Ping(context.Background()); err != nil {
				t.Errorf("database %s should be connected", name)
			}
		}
	})

	t.Run("partial connection failures", func(t *testing.T) {
		t.Parallel()
		pool := newDatabasePool()

		goodDB := NewDatabase("sqlite3", ":memory:")
		badDB := NewDatabase("invalid_driver", "invalid_dsn")

		_ = pool.registerDatabase("good", goodDB)
		_ = pool.registerDatabase("bad", badDB)

		err := pool.connectAll()
		if err == nil {
			t.Error("expected error for failed connections")
		}
		if !errors.Is(err, ErrMultipleFailedToOpenDatabase) {
			t.Errorf("expected ErrMultipleFailedToOpenDatabase, got %v", err)
		}
	})

	t.Run("empty pool", func(t *testing.T) {
		t.Parallel()
		pool := newDatabasePool()

		err := pool.connectAll()
		if err != nil {
			t.Errorf("connectAll should succeed with empty pool: %v", err)
		}
	})
}

func TestCloseAll(t *testing.T) {
	t.Parallel()

	t.Run("close all connections", func(t *testing.T) {
		t.Parallel()
		pool := newDatabasePool()

		dbs := make([]contracts.Database, 3)
		for i := 0; i < 3; i++ {
			db := NewDatabase("sqlite3", ":memory:")
			_ = db.Connect()
			dbs[i] = db
			_ = pool.registerDatabase(string(rune('a'+i)), db)
		}

		err := pool.closeAll()
		if err != nil {
			t.Errorf("closeAll failed: %v", err)
		}

		for i, db := range dbs {
			if err := db.Ping(context.Background()); err == nil {
				t.Errorf("database %d should be closed", i)
			}
		}
	})

	t.Run("handle close errors", func(t *testing.T) {
		t.Parallel()
		pool := newDatabasePool()

		db1 := NewDatabase("sqlite3", ":memory:")
		_ = db1.Connect()
		_ = pool.registerDatabase("db1", db1)

		db2 := NewDatabase("sqlite3", ":memory:")
		_ = pool.registerDatabase("db2", db2)

		err := pool.closeAll()
		if err != nil {
			t.Logf("closeAll with mixed states: %v", err)
		}
	})

	t.Run("empty pool", func(t *testing.T) {
		t.Parallel()
		pool := newDatabasePool()

		err := pool.closeAll()
		if err != nil {
			t.Errorf("closeAll should succeed with empty pool: %v", err)
		}
	})
}

func TestGetConnectionNames(t *testing.T) {
	t.Parallel()

	t.Run("sorted names", func(t *testing.T) {
		t.Parallel()
		pool := newDatabasePool()

		names := []string{"zebra", "alpha", "beta", "gamma"}
		for _, name := range names {
			db := NewDatabase("sqlite3", ":memory:")
			_ = pool.registerDatabase(name, db)
		}

		gotNames := pool.getConnectionNames()

		if len(gotNames) != len(names) {
			t.Errorf("expected %d names, got %d", len(names), len(gotNames))
		}

		expected := []string{"alpha", "beta", "gamma", "zebra"}
		for i, name := range gotNames {
			if name != expected[i] {
				t.Errorf("expected name[%d] to be %s, got %s", i, expected[i], name)
			}
		}
	})

	t.Run("empty pool", func(t *testing.T) {
		t.Parallel()
		pool := newDatabasePool()

		names := pool.getConnectionNames()
		if len(names) != 0 {
			t.Errorf("expected empty slice, got %d names", len(names))
		}
	})

	t.Run("concurrent access", func(t *testing.T) {
		t.Parallel()
		pool := newDatabasePool()

		for i := 0; i < 5; i++ {
			db := NewDatabase("sqlite3", ":memory:")
			_ = pool.registerDatabase(string(rune('a'+i)), db)
		}

		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				names := pool.getConnectionNames()
				if len(names) != 5 {
					t.Errorf("expected 5 names, got %d", len(names))
				}
			}()
		}
		wg.Wait()
	})
}

func TestDatabasePoolRaceConditions(t *testing.T) {
	t.Run("concurrent operations", func(t *testing.T) {
		pool := newDatabasePool()
		var wg sync.WaitGroup

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				name := string(rune('a' + id))
				db := NewDatabase("sqlite3", ":memory:")

				_ = pool.registerDatabase(name, db)

				_, _ = pool.getDatabase(name)

				_ = pool.getConnectionNames()
			}(i)
		}

		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = pool.connectAll()
			}()
		}

		wg.Wait()

		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = pool.closeAll()
		}()

		wg.Wait()
	})
}

func TestDatabasePoolIntegration(t *testing.T) {
	t.Run("full lifecycle", func(t *testing.T) {
		pool := newDatabasePool()

		db1 := NewDatabase("sqlite3", ":memory:")
		db2 := NewDatabase("sqlite3", ":memory:")
		db3 := NewDatabase("sqlite3", ":memory:")

		if err := pool.registerDatabase("primary", db1); err != nil {
			t.Fatalf("failed to register primary: %v", err)
		}
		if err := pool.registerDatabase("secondary", db2); err != nil {
			t.Fatalf("failed to register secondary: %v", err)
		}
		if err := pool.registerDatabase("tertiary", db3); err != nil {
			t.Fatalf("failed to register tertiary: %v", err)
		}

		if err := pool.connectAll(); err != nil {
			t.Fatalf("failed to connect all: %v", err)
		}

		names := pool.getConnectionNames()
		if len(names) != 3 {
			t.Errorf("expected 3 connections, got %d", len(names))
		}

		for _, name := range names {
			db, exists := pool.getDatabase(name)
			if !exists {
				t.Errorf("database %s should exist", name)
				continue
			}
			if err := db.Ping(context.Background()); err != nil {
				t.Errorf("database %s should be connected: %v", name, err)
			}
		}

		if err := pool.closeAll(); err != nil {
			t.Errorf("failed to close all: %v", err)
		}
	})
}

func TestDatabasePoolErrorMessages(t *testing.T) {
	t.Parallel()

	t.Run("detailed error messages", func(t *testing.T) {
		t.Parallel()
		pool := newDatabasePool()
		db := NewDatabase("sqlite3", ":memory:")

		_ = pool.registerDatabase("test", db)
		err := pool.registerDatabase("test", db)

		if err == nil {
			t.Fatal("expected error")
		}

		errStr := err.Error()
		if !containsAll(errStr, "test", "connection already exists") {
			t.Errorf("error message should contain connection name and reason: %s", errStr)
		}
	})
}

func containsAll(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if !contains(s, substr) {
			return false
		}
	}
	return true
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && hasSubstring(s, substr)
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
