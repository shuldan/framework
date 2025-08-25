package config

import (
	"os"
	"reflect"
	"testing"
)

func TestEnvConfigLoader_Load(t *testing.T) {
	os.Setenv("APP_NAME", "testapp")
	os.Setenv("APP_PORT", "8080")
	os.Setenv("APP_FEATURES__ENABLED", "true")
	os.Setenv("APP_DB__HOST", "localhost")
	os.Setenv("APP_SLICE", "a,b,c")
	os.Setenv("PREFIX_IGNORED", "x")
	defer func() {
		os.Unsetenv("APP_NAME")
		os.Unsetenv("APP_PORT")
		os.Unsetenv("APP_FEATURES__ENABLED")
		os.Unsetenv("APP_DB__HOST")
		os.Unsetenv("APP_SLICE")
		os.Unsetenv("PREFIX_IGNORED")
	}()

	loader := &EnvConfigLoader{prefix: "APP_"}
	config, err := loader.Load()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := map[string]any{
		"name": "testapp",
		"port": 8080,
		"features": map[string]any{
			"enabled": true,
		},
		"db": map[string]any{
			"host": "localhost",
		},
		"slice": "a,b,c",
	}

	if !reflect.DeepEqual(config, expected) {
		t.Errorf("got %v, expected %v", config, expected)
	}
}

func TestEnvConfigLoader_Load_EmptyPrefix(t *testing.T) {
	loader := &EnvConfigLoader{prefix: ""}
	config, err := loader.Load()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if config == nil {
		t.Error("expected non-nil config")
	}
}
