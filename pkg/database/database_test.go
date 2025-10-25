package database

import (
	"context"
	"errors"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/shuldan/framework/pkg/contracts"
)

func TestNewDatabase(t *testing.T) {
	tests := []struct {
		name    string
		driver  string
		dsn     string
		options []Option
	}{
		{
			name:   "basic database creation",
			driver: "sqlite3",
			dsn:    ":memory:",
		},
		{
			name:   "database with connection pool options",
			driver: "sqlite3",
			dsn:    ":memory:",
			options: []Option{
				WithConnectionPool(10, 5, time.Hour),
				WithPingTimeout(time.Second * 10),
			},
		},
		{
			name:   "database with retry options",
			driver: "sqlite3",
			dsn:    ":memory:",
			options: []Option{
				WithRetry(5, time.Millisecond*100),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := NewDatabase(tt.driver, tt.dsn, tt.options...)
			if db == nil {
				t.Fatal("NewDatabase returned nil")
			}

			sqlDB := db.(*sqlDatabase)
			if sqlDB.driver != tt.driver {
				t.Errorf("expected driver %s, got %s", tt.driver, sqlDB.driver)
			}
			if sqlDB.dsn != tt.dsn {
				t.Errorf("expected dsn %s, got %s", tt.dsn, sqlDB.dsn)
			}
		})
	}
}

func TestDatabaseConnect(t *testing.T) {
	tests := []struct {
		name        string
		driver      string
		dsn         string
		expectError bool
	}{
		{
			name:   "successful connection",
			driver: "sqlite3",
			dsn:    ":memory:",
		},
		{
			name:        "invalid driver",
			driver:      "invalid",
			dsn:         ":memory:",
			expectError: true,
		},
		{
			name:        "invalid dsn",
			driver:      "sqlite3",
			dsn:         "/invalid/path/db.sqlite",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := NewDatabase(tt.driver, tt.dsn)
			err := db.Connect()

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectError {
				err2 := db.Connect()
				if err2 != nil {
					t.Errorf("second connect failed: %v", err2)
				}

				ctx := context.Background()
				if err := db.Ping(ctx); err != nil {
					t.Errorf("ping failed: %v", err)
				}

				if err := db.Close(); err != nil {
					t.Errorf("failed to close database: %v", err)
				}
			}
		})
	}
}

func TestDatabaseOperationsWithoutConnection(t *testing.T) {
	db := NewDatabase("sqlite3", ":memory:")

	ctx := context.Background()

	err := db.Ping(ctx)
	if !errors.Is(err, ErrDatabaseNotConnected) {
		t.Errorf("expected ErrDatabaseNotConnected, got %v", err)
	}

	err = db.Migrate([]contracts.Migration{})
	if !errors.Is(err, ErrDatabaseNotConnected) {
		t.Errorf("expected ErrDatabaseNotConnected, got %v", err)
	}
}

func TestDatabaseOptions(t *testing.T) {
	tests := []struct {
		name     string
		option   Option
		validate func(*dbConfig) bool
	}{
		{
			name:   "connection pool option",
			option: WithConnectionPool(20, 10, time.Hour*2),
			validate: func(config *dbConfig) bool {
				return config.maxOpenConns == 20 && config.maxIdleConns == 10 && config.connMaxLifetime == time.Hour*2
			},
		},
		{
			name:   "connection idle time option",
			option: WithConnectionIdleTime(time.Minute * 10),
			validate: func(config *dbConfig) bool {
				return config.connMaxIdleTime == time.Minute*10
			},
		},
		{
			name:   "ping timeout option",
			option: WithPingTimeout(time.Second * 30),
			validate: func(config *dbConfig) bool {
				return config.pingTimeout == time.Second*30
			},
		},
		{
			name:   "retry option",
			option: WithRetry(10, time.Second*2),
			validate: func(config *dbConfig) bool {
				return config.retryAttempts == 10 && config.retryDelay == time.Second*2
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &dbConfig{}
			tt.option(config)

			if !tt.validate(config) {
				t.Error("option validation failed")
			}
		})
	}
}

func setupTestDatabase(t *testing.T) contracts.Database {
	tempFile := t.TempDir() + "/test.db"
	db := NewDatabase(
		"sqlite3",
		tempFile,
		WithConnectionPool(1, 1, time.Minute),
		WithRetry(5, time.Millisecond*100),
	)
	if err := db.Connect(); err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	sqlDB := db.(*sqlDatabase).db
	_, err := sqlDB.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		t.Fatalf("failed to set journal_mode=WAL: %v", err)
	}

	var mode string
	err = sqlDB.QueryRow("PRAGMA journal_mode;").Scan(&mode)
	if err != nil || mode != "wal" {
		t.Fatalf("WAL not enabled, mode is: %s, err: %v", mode, err)
	}

	return db
}
