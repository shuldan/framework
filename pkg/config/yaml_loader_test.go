package config

import (
	"errors"
	"os"
	"testing"
)

func TestYamlConfigLoader_Load_Success(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "config*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := `
app:
  name: yamlapp
  port: 8081
debug: false
features:
  new_ui: true
`
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	loader := NewYamlConfigLoader(tmpfile.Name())
	config, err := loader.Load()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg := NewMapConfig(config)
	if cfg.GetInt("app.port") != 8081 {
		t.Errorf("expected app.port = 8081, got %v", cfg.GetInt("app.port"))
	}
	if cfg.GetString("app.name") != "yamlapp" {
		t.Errorf("expected app.name = 'yamlapp', got %v", cfg.GetString("app.name"))
	}
	if cfg.GetBool("debug") != false {
		t.Errorf("expected debug = false, got %v", cfg.GetBool("debug"))
	}
	if cfg.GetBool("features.new_ui") != true {
		t.Errorf("expected features.new_ui = true, got %v", cfg.GetBool("features.new_ui"))
	}
}

func TestYamlConfigLoader_Load_FileNotFound(t *testing.T) {
	loader := NewYamlConfigLoader("nonexistent.yaml")
	_, err := loader.Load()

	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !errors.Is(err, ErrNoConfigSource) {
		t.Errorf("expected ErrNoConfigSource, got %v", err)
	}
}
