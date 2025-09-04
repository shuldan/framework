package config

import (
	"errors"
	"os"
	"reflect"
	"testing"
)

func TestTemplatedLoader_Load_BasicCases(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]any
		envVars  map[string]string
		expected map[string]any
	}{
		{
			name: "basic_templating",
			config: map[string]any{
				"host": "{{.HOST}}",
				"port": "{{.PORT}}",
				"name": "test-app",
			},
			envVars: map[string]string{
				"HOST": "localhost",
				"PORT": "8080",
			},
			expected: map[string]any{
				"host": "localhost",
				"port": "8080",
				"name": "test-app",
			},
		},
		{
			name: "non_template_strings",
			config: map[string]any{
				"normal_string": "no templates here",
				"partial":       "prefix {{.VAR}} suffix",
				"number":        42,
				"boolean":       true,
			},
			envVars: map[string]string{
				"VAR": "middle",
			},
			expected: map[string]any{
				"normal_string": "no templates here",
				"partial":       "prefix middle suffix",
				"number":        42,
				"boolean":       true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runTemplateTest(t, tt.config, tt.envVars, tt.expected)
		})
	}
}

func TestTemplatedLoader_Load_NestedStructures(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]any
		envVars  map[string]string
		expected map[string]any
	}{
		{
			name: "nested_maps",
			config: map[string]any{
				"database": map[string]any{
					"host": "{{.DB_HOST}}",
					"port": "{{.DB_PORT}}",
				},
				"cache": map[string]any{
					"enabled": true,
				},
			},
			envVars: map[string]string{
				"DB_HOST": "db.example.com",
				"DB_PORT": "5432",
			},
			expected: map[string]any{
				"database": map[string]any{
					"host": "db.example.com",
					"port": "5432",
				},
				"cache": map[string]any{
					"enabled": true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runTemplateTest(t, tt.config, tt.envVars, tt.expected)
		})
	}
}

func TestTemplatedLoader_Load_ArrayProcessing(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]any
		envVars  map[string]string
		expected map[string]any
	}{
		{
			name: "array_processing",
			config: map[string]any{
				"servers": []any{
					"{{.SERVER1}}",
					"{{.SERVER2}}",
					"static-server",
				},
			},
			envVars: map[string]string{
				"SERVER1": "srv1.com",
				"SERVER2": "srv2.com",
			},
			expected: map[string]any{
				"servers": []any{"srv1.com", "srv2.com", "static-server"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runTemplateTest(t, tt.config, tt.envVars, tt.expected)
		})
	}
}

func TestTemplatedLoader_Load_TemplateFunctions(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]any
		envVars  map[string]string
		expected map[string]any
	}{
		{
			name: "template_functions",
			config: map[string]any{
				"env_var": "{{ env \"TEST_VAR\" }}",
				"upper":   "{{ upper \"hello\" }}",
				"lower":   "{{ lower \"WORLD\" }}",
				"default": "{{ default \"fallback\" .MISSING_VAR }}",
			},
			envVars: map[string]string{
				"TEST_VAR": "test_value",
			},
			expected: map[string]any{
				"env_var": "test_value",
				"upper":   "HELLO",
				"lower":   "world",
				"default": "fallback",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runTemplateTest(t, tt.config, tt.envVars, tt.expected)
		})
	}
}

func TestTemplatedLoader_Load_InvalidTemplates(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]any
		envVars  map[string]string
		expected map[string]any
	}{
		{
			name: "invalid_template",
			config: map[string]any{
				"invalid": "{{ .VAR",
				"valid":   "{{.VALID_VAR}}",
			},
			envVars: map[string]string{
				"VALID_VAR": "works",
			},
			expected: map[string]any{
				"invalid": "{{ .VAR",
				"valid":   "works",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runTemplateTest(t, tt.config, tt.envVars, tt.expected)
		})
	}
}

func TestTemplatedLoader_Load_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		loader      Loader
		expectedErr error
	}{
		{
			name:        "loader_error",
			loader:      &mockLoader{err: ErrNoConfigSource},
			expectedErr: ErrNoConfigSource,
		},
		{
			name: "nil_config",
			loader: &mockLoader{
				config: nil,
				err:    nil,
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			templatedLoader := newTemplatedLoader(tt.loader)
			result, err := templatedLoader.Load()

			if tt.expectedErr != nil {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !errors.Is(err, tt.expectedErr) {
					t.Errorf("expected error %v, got %v", tt.expectedErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if tt.loader.(*mockLoader).config == nil && result == nil {
				t.Error("expected empty map for nil config, got nil")
			}
		})
	}
}

func TestTemplatedLoader_ProcessValue_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		envVars  map[string]string
		expected any
	}{
		{
			name:     "nil_value",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty_string",
			input:    "",
			expected: "",
		},
		{
			name:     "string_with_no_templates",
			input:    "plain text",
			expected: "plain text",
		},
		{
			name:     "string_with_partial_template_markers",
			input:    "{{ incomplete",
			expected: "{{ incomplete",
		},
		{
			name:     "nested_empty_map",
			input:    map[string]any{},
			expected: map[string]any{},
		},
		{
			name:     "nested_empty_array",
			input:    []any{},
			expected: []any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runEdgeCaseTest(t, tt.input, tt.envVars, tt.expected)
		})
	}
}

func runTemplateTest(t *testing.T, config map[string]any, envVars map[string]string, expected map[string]any) {
	setEnvVars(t, envVars)
	defer unsetEnvVars(t, envVars)

	mockLoader := &mockLoader{config: config}
	templatedLoader := newTemplatedLoader(mockLoader)

	result, err := templatedLoader.Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func runEdgeCaseTest(t *testing.T, input any, envVars map[string]string, expected any) {
	setEnvVars(t, envVars)
	defer unsetEnvVars(t, envVars)

	mockLoader := &mockLoader{
		config: map[string]any{"test": input},
	}
	templatedLoader := newTemplatedLoader(mockLoader)

	result, err := templatedLoader.Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	actual := result["test"]

	if isEmptySlice(expected) && isEmptySlice(actual) {
		return
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("expected %v, got %v", expected, actual)
	}
}

func isEmptySlice(v any) bool {
	if v == nil {
		return false
	}
	val := reflect.ValueOf(v)
	return val.Kind() == reflect.Slice && val.Len() == 0
}

func setEnvVars(t *testing.T, envVars map[string]string) {
	for key, value := range envVars {
		if err := os.Setenv(key, value); err != nil {
			t.Fatalf("failed to set %s: %v", key, err)
		}
	}
}

func unsetEnvVars(t *testing.T, envVars map[string]string) {
	for key := range envVars {
		if err := os.Unsetenv(key); err != nil {
			t.Errorf("failed to unset %s: %v", key, err)
		}
	}
}
