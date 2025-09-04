package config

import (
	"os"
	"reflect"
	"testing"
)

func TestEnvConfigLoader_Load(t *testing.T) {
	if err := os.Setenv("APP_NAME", "testapp"); err != nil {
		t.Fatalf("failed to set APP_NAME: %v", err)
	}
	if err := os.Setenv("APP_PORT", "8080"); err != nil {
		t.Fatalf("failed to set APP_PORT: %v", err)
	}
	if err := os.Setenv("APP_FEATURES__ENABLED", "true"); err != nil {
		t.Fatalf("failed to set APP_FEATURES__ENABLED: %v", err)
	}
	if err := os.Setenv("APP_DB__HOST", "localhost"); err != nil {
		t.Fatalf("failed to set APP_DB__HOST: %v", err)
	}
	if err := os.Setenv("APP_SLICE", "a,b,c"); err != nil {
		t.Fatalf("failed to set APP_SLICE: %v", err)
	}
	if err := os.Setenv("PREFIX_IGNORED", "x"); err != nil {
		t.Fatalf("failed to set PREFIX_IGNORED: %v", err)
	}
	defer func() {
		for _, key := range []string{
			"APP_NAME",
			"APP_PORT",
			"APP_FEATURES__ENABLED",
			"APP_DB__HOST",
			"APP_SLICE",
			"PREFIX_IGNORED",
		} {
			if val, exists := os.LookupEnv(key); exists {
				if err := os.Unsetenv(key); err != nil {
					t.Logf("failed to unset %s (value: %s): %v", key, val, err)
				}
			}
		}
	}()

	loader := &envConfigLoader{prefix: "APP_"}
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
	loader := &envConfigLoader{prefix: ""}
	config, err := loader.Load()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if config == nil {
		t.Error("expected non-nil config")
	}
}
