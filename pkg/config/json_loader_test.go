package config

import (
	"errors"
	"os"
	"testing"
)

func TestJSONConfigLoader_Load_Success(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "config*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tmpfile.Name()); err != nil {
			t.Logf("failed to remove temp file %s: %v", tmpfile.Name(), err)
		}
	}()

	content := `{"app": {"name": "jsonapp", "port": 8080}, "debug": true}`
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	loader := NewJSONConfigLoader(tmpfile.Name())
	config, err := loader.Load()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	app := config["app"].(map[string]any)
	if app["name"].(string) != "jsonapp" {
		t.Errorf("expected app.name = 'jsonapp', got %v", app["name"])
	}
	if app["port"].(float64) != 8080 {
		t.Errorf("expected app.port = 8080, got %v", app["port"])
	}
	if config["debug"] != true {
		t.Errorf("expected debug = true, got %v", config["debug"])
	}
}

func TestJSONConfigLoader_Load_FileNotFound(t *testing.T) {
	loader := NewJSONConfigLoader("nonexistent.json")
	_, err := loader.Load()

	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !errors.Is(err, ErrNoConfigSource) {
		t.Errorf("expected ErrNoConfigSource, got %v", err)
	}
}

func TestJSONConfigLoader_Load_InvalidJSON(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "invalid*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tmpfile.Name()); err != nil {
			t.Logf("failed to remove temp file %s: %v", tmpfile.Name(), err)
		}
	}()

	if _, err := tmpfile.Write([]byte("{invalid json}")); err != nil {
		t.Fatalf("failed to write invalid JSON: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	loader := NewJSONConfigLoader(tmpfile.Name())
	_, err = loader.Load()

	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !errors.Is(err, ErrParseJSON) {
		t.Errorf("expected ErrParseJSON, got %v", err)
	}
}
