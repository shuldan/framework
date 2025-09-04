package config

import (
	"errors"
	"testing"
)

type mockLoader struct {
	config map[string]any
	err    error
}

func (m *mockLoader) Load() (map[string]any, error) {
	return m.config, m.err
}

func TestChainLoader_Load_SuccessfulMerge(t *testing.T) {
	loader1 := &mockLoader{
		config: map[string]any{
			"app": map[string]any{
				"name": "test",
				"port": 8080,
			},
		},
		err: nil,
	}

	loader2 := &mockLoader{
		config: map[string]any{
			"app": map[string]any{
				"port": 9000,
				"env":  "prod",
			},
			"db": "localhost",
		},
		err: nil,
	}

	chain := &chainLoader{loaders: []Loader{loader1, loader2}}
	result, err := chain.Load()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	app, ok := result["app"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'app' to be map[string]any")
	}
	if app["name"] != "test" {
		t.Errorf("expected app.name = 'test', got %v", app["name"])
	}
	if app["port"] != 9000 {
		t.Errorf("expected app.port = 9000, got %v", app["port"])
	}
	if app["env"] != "prod" {
		t.Errorf("expected app.env = 'prod', got %v", app["env"])
	}
	if result["db"] != "localhost" {
		t.Errorf("expected db = 'localhost', got %v", result["db"])
	}
}

func TestChainLoader_Load_ErrorInLoader(t *testing.T) {
	loader1 := &mockLoader{err: errors.New("failed to Load")}
	loader2 := &mockLoader{
		config: map[string]any{"key": "value"},
		err:    nil,
	}

	chain := &chainLoader{loaders: []Loader{loader1, loader2}}
	result, err := chain.Load()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("expected key = 'value', got %v", result["key"])
	}
}

func TestChainLoader_Load_AllLoadersFail(t *testing.T) {
	loader1 := &mockLoader{err: errors.New("err1")}
	loader2 := &mockLoader{err: errors.New("err2")}

	chain := &chainLoader{loaders: []Loader{loader1, loader2}}
	_, err := chain.Load()

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNoConfigSource) {
		t.Errorf("expected ErrNoConfigSource, got %v", err)
	}
}

func TestChainLoader_Load_OverrideWithScalar(t *testing.T) {
	loader1 := &mockLoader{
		config: map[string]any{
			"nested": map[string]any{"key": "value"},
		},
		err: nil,
	}

	loader2 := &mockLoader{
		config: map[string]any{
			"nested": "string_value",
		},
		err: nil,
	}

	chain := &chainLoader{loaders: []Loader{loader1, loader2}}
	result, err := chain.Load()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result["nested"] != "string_value" {
		t.Errorf("expected nested = 'string_value', got %v", result["nested"])
	}
}
