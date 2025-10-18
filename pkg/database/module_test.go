package database

import (
	"reflect"
	"testing"
	"time"

	"github.com/shuldan/framework/pkg/app"
	"github.com/shuldan/framework/pkg/config"
	"github.com/shuldan/framework/pkg/contracts"
	"github.com/shuldan/framework/pkg/errors"
)

func TestModule_Register(t *testing.T) {
	cfg := config.NewMapConfig(map[string]interface{}{
		"database": map[string]interface{}{
			"default": "primary",
			"connections": map[string]interface{}{
				"primary": map[string]interface{}{
					"driver": "sqlite3",
					"dsn":    ":memory:",
					"pool": map[string]interface{}{
						"max_open_connections": 10,
						"max_idle_connections": 5,
						"conn_max_lifetime":    "1h",
						"conn_max_idle_time":   "5m",
					},
				},
			},
		},
	})

	container := app.NewContainer()

	err := container.Instance(reflect.TypeOf((*contracts.Config)(nil)).Elem(), cfg)
	if err != nil {
		t.Fatalf("failed to register dbConfig: %v", err)
	}

	m := NewModule()
	err = m.Register(container)
	if err != nil {
		t.Fatalf("failed to register module: %v", err)
	}
	err = m.Register(container)
	if err == nil {
		t.Fatalf("cannot register module twice")
	}

	if !container.Has(reflect.TypeOf((*contracts.Database)(nil)).Elem()) {
		t.Error("database should be registered")
	}
}

func TestModule_CreateConnection(t *testing.T) {
	m := &module{
		pool: newDatabasePool(),
	}

	tests := []struct {
		name        string
		config      map[string]interface{}
		expectError bool
		errorType   error
	}{
		{
			name: "valid sqlite connection",
			config: map[string]interface{}{
				"driver": "sqlite",
				"dsn":    ":memory:",
			},
			expectError: false,
		},
		{
			name: "missing driver",
			config: map[string]interface{}{
				"dsn": ":memory:",
			},
			expectError: true,
			errorType:   ErrDriverNotSpecified,
		},
		{
			name: "missing dsn",
			config: map[string]interface{}{
				"driver": "sqlite",
			},
			expectError: true,
			errorType:   ErrDSNNotSpecified,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := m.createConnection("test", tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				if tt.errorType != nil && !errors.Is(err, tt.errorType) {
					t.Errorf("expected error %v, got %v", tt.errorType, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if db == nil {
					t.Error("database should not be nil")
				}
			}
		})
	}
}

func TestModule_GetSQLDriver(t *testing.T) {
	m := &module{}

	tests := []struct {
		input    string
		expected string
	}{
		{"mysql", "mysql"},
		{"MySQL", "mysql"},
		{"postgres", "postgres"},
		{"postgresql", "postgres"},
		{"PostgreSQL", "postgres"},
		{"sqlite", "sqlite3"},
		{"sqlite3", "sqlite3"},
		{"SQLite", "sqlite3"},
		{"custom", "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := m.getSQLDriver(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestModule_GetIntValue(t *testing.T) {
	m := &module{}
	tests := []struct {
		name         string
		config       map[string]interface{}
		key          string
		defaultValue int
		expected     int
	}{
		{
			name:         "int value",
			config:       map[string]interface{}{"max": 100},
			key:          "max",
			defaultValue: 10,
			expected:     100,
		},
		{
			name:         "int64 value",
			config:       map[string]interface{}{"max": int64(200)},
			key:          "max",
			defaultValue: 10,
			expected:     200,
		},
		{
			name:         "float64 value",
			config:       map[string]interface{}{"max": float64(300)},
			key:          "max",
			defaultValue: 10,
			expected:     300,
		},
		{
			name:         "string value",
			config:       map[string]interface{}{"max": "400"},
			key:          "max",
			defaultValue: 10,
			expected:     400,
		},
		{
			name:         "missing key",
			config:       map[string]interface{}{},
			key:          "max",
			defaultValue: 10,
			expected:     10,
		},
		{
			name:         "invalid string",
			config:       map[string]interface{}{"max": "invalid"},
			key:          "max",
			defaultValue: 10,
			expected:     10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.getIntValue(tt.config, tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestModule_GetDurationValue(t *testing.T) {
	m := &module{}
	tests := []struct {
		name         string
		config       map[string]interface{}
		key          string
		defaultValue time.Duration
		expected     time.Duration
	}{
		{
			name:         "string duration",
			config:       map[string]interface{}{"timeout": "5m"},
			key:          "timeout",
			defaultValue: time.Hour,
			expected:     5 * time.Minute,
		},
		{
			name:         "int seconds",
			config:       map[string]interface{}{"timeout": 60},
			key:          "timeout",
			defaultValue: time.Hour,
			expected:     60 * time.Second,
		},
		{
			name:         "int64 seconds",
			config:       map[string]interface{}{"timeout": int64(120)},
			key:          "timeout",
			defaultValue: time.Hour,
			expected:     120 * time.Second,
		},
		{
			name:         "float64 seconds",
			config:       map[string]interface{}{"timeout": float64(180)},
			key:          "timeout",
			defaultValue: time.Hour,
			expected:     180 * time.Second,
		},
		{
			name:         "missing key",
			config:       map[string]interface{}{},
			key:          "timeout",
			defaultValue: time.Hour,
			expected:     time.Hour,
		},
		{
			name:         "invalid string",
			config:       map[string]interface{}{"timeout": "invalid"},
			key:          "timeout",
			defaultValue: time.Hour,
			expected:     time.Hour,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.getDurationValue(tt.config, tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
