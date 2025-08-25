package config

import (
	"errors"
	"io/ioutil"
	"os"
	"testing"
)

func TestJSONConfigLoader_Load_Success(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "config*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := `{"app": {"name": "jsonapp", "port": 8080}, "debug": true}`
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	loader := NewJSONConfigLoader(tmpfile.Name())
	config, err := loader.Load()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config["app"].(map[string]any)["name"] != "jsonapp" {
		t.Errorf("expected app.name = 'jsonapp'")
	}
	if config["app"].(map[string]any)["port"].(float64) != 8080 {
		t.Errorf("expected app.port = 8080")
	}
	if config["debug"] != true {
		t.Errorf("expected debug = true")
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
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte("{invalid json}")); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	loader := NewJSONConfigLoader(tmpfile.Name())
	_, err = loader.Load()

	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !errors.Is(err, ErrParseJSON) {
		t.Errorf("expected ErrParseJSON, got %v", err)
	}
}
