package config

import (
	"reflect"
	"testing"
)

func TestMapConfig_Get(t *testing.T) {
	cfg := NewMapConfig(map[string]any{
		"app": map[string]any{
			"port": 8080,
		},
		"features": map[string]any{
			"new_ui": true,
		},
	})

	if cfg.Get("app.port") != 8080 {
		t.Errorf("expected app.port = 8080, got %v", cfg.Get("app.port"))
	}
	if cfg.Get("features.new_ui") != true {
		t.Errorf("expected features.new_ui = true, got %v", cfg.Get("features.new_ui"))
	}
	if cfg.Get("unknown") != nil {
		t.Errorf("expected unknown = nil, got %v", cfg.Get("unknown"))
	}
}

func TestMapConfig_Has(t *testing.T) {
	cfg := NewMapConfig(map[string]any{
		"app": map[string]any{
			"name": "myapp",
		},
		"db": nil,
	})

	if !cfg.Has("app.name") {
		t.Error("expected Has('app.name') = true")
	}
	if !cfg.Has("db") {
		t.Error("expected Has('db') = true (even if nil)")
	}
	if cfg.Has("missing") {
		t.Error("expected Has('missing') = false")
	}
}

func TestMapConfig_GetString(t *testing.T) {
	cfg := NewMapConfig(map[string]any{
		"string": "hello",
		"int":    42,
		"bool":   true,
		"float":  3.14,
		"nil":    nil,
	})

	tests := []struct {
		key      string
		expected string
	}{
		{"string", "hello"},
		{"int", "42"},
		{"bool", "true"},
		{"float", "3.14"},
		{"nil", ""},
		{"missing", "default"},
	}

	for _, tt := range tests {
		got := cfg.GetString(tt.key, "default")
		if got != tt.expected {
			t.Errorf("GetString(%s) = %q, expected %q", tt.key, got, tt.expected)
		}
	}
}

func TestMapConfig_GetInt(t *testing.T) {
	cfg := NewMapConfig(map[string]any{
		"int":    42,
		"float":  3.14,
		"string": "123",
		"bool":   true,
	})

	if cfg.GetInt("int") != 42 {
		t.Errorf("GetInt('int') = %d, expected 42", cfg.GetInt("int"))
	}
	if cfg.GetInt("float") != 3 {
		t.Errorf("GetInt('float') = %d, expected 3", cfg.GetInt("float"))
	}
	if cfg.GetInt("string") != 123 {
		t.Errorf("GetInt('string') = %d, expected 123", cfg.GetInt("string"))
	}
	if cfg.GetInt("bool") != 1 {
		t.Errorf("GetInt('bool') = %d, expected 1", cfg.GetInt("bool"))
	}
	if cfg.GetInt("missing", 99) != 99 {
		t.Errorf("GetInt('missing') = %d, expected 99", cfg.GetInt("missing", 99))
	}
}

func TestMapConfig_GetInt64(t *testing.T) {
	cfg := NewMapConfig(map[string]any{
		"int64":  int64(1000),
		"int":    100,
		"float":  99.9,
		"string": "500",
	})

	if cfg.GetInt64("int64") != 1000 {
		t.Errorf("GetInt64('int64') = %d, expected 1000", cfg.GetInt64("int64"))
	}
	if cfg.GetInt64("int") != 100 {
		t.Errorf("GetInt64('int') = %d, expected 100", cfg.GetInt64("int"))
	}
	if cfg.GetInt64("float") != 99 {
		t.Errorf("GetInt64('float') = %d, expected 99", cfg.GetInt64("float"))
	}
	if cfg.GetInt64("string") != 500 {
		t.Errorf("GetInt64('string') = %d, expected 500", cfg.GetInt64("string"))
	}
	if cfg.GetInt64("missing", 42) != 42 {
		t.Errorf("GetInt64('missing') = %d, expected 42", cfg.GetInt64("missing", 42))
	}
}

func TestMapConfig_GetFloat64(t *testing.T) {
	cfg := NewMapConfig(map[string]any{
		"float":  3.14,
		"int":    42,
		"int64":  int64(100),
		"string": "2.5",
	})

	if cfg.GetFloat64("float") != 3.14 {
		t.Errorf("GetFloat64('float') = %f, expected 3.14", cfg.GetFloat64("float"))
	}
	if cfg.GetFloat64("int") != 42.0 {
		t.Errorf("GetFloat64('int') = %f, expected 42.0", cfg.GetFloat64("int"))
	}
	if cfg.GetFloat64("int64") != 100.0 {
		t.Errorf("GetFloat64('int64') = %f, expected 100.0", cfg.GetFloat64("int64"))
	}
	if cfg.GetFloat64("string") != 2.5 {
		t.Errorf("GetFloat64('string') = %f, expected 2.5", cfg.GetFloat64("string"))
	}
	if cfg.GetFloat64("missing", 1.1) != 1.1 {
		t.Errorf("GetFloat64('missing') = %f, expected 1.1", cfg.GetFloat64("missing", 1.1))
	}
}

func TestMapConfig_GetBool(t *testing.T) {
	cfg := NewMapConfig(map[string]any{
		"bool":         true,
		"string_true":  "true",
		"string_yes":   "yes",
		"string_1":     "1",
		"string_false": "false",
		"int":          1,
		"zero":         0,
	})

	tests := []struct {
		key      string
		expected bool
	}{
		{"bool", true},
		{"string_true", true},
		{"string_yes", true},
		{"string_1", true},
		{"int", true},
		{"string_false", false},
		{"zero", false},
		{"missing", true},
	}

	for _, tt := range tests {
		defaultVal := tt.expected
		got := cfg.GetBool(tt.key, defaultVal)
		if got != tt.expected {
			t.Errorf("GetBool(%s) = %t, expected %t", tt.key, got, tt.expected)
		}
	}
}

func TestMapConfig_GetStringSlice(t *testing.T) {
	cfg := NewMapConfig(map[string]any{
		"comma":     "a,b,c",
		"custom":    "x|y|z",
		"array":     []string{"p", "q", "r"},
		"mixed":     []any{1, "two", 3.0},
		"single":    "single",
		"not_found": nil,
	})

	if !reflect.DeepEqual(cfg.GetStringSlice("comma"), []string{"a", "b", "c"}) {
		t.Errorf("GetStringSlice('comma') = %v", cfg.GetStringSlice("comma"))
	}

	if !reflect.DeepEqual(cfg.GetStringSlice("custom", "|"), []string{"x", "y", "z"}) {
		t.Errorf("GetStringSlice('custom', '|') = %v", cfg.GetStringSlice("custom", "|"))
	}

	if !reflect.DeepEqual(cfg.GetStringSlice("array"), []string{"p", "q", "r"}) {
		t.Errorf("GetStringSlice('array') = %v", cfg.GetStringSlice("array"))
	}

	if !reflect.DeepEqual(cfg.GetStringSlice("mixed"), []string{"1", "two", "3"}) {
		t.Errorf("GetStringSlice('mixed') = %v", cfg.GetStringSlice("mixed"))
	}

	if !reflect.DeepEqual(cfg.GetStringSlice("single"), []string{"single"}) {
		t.Errorf("GetStringSlice('single') = %v", cfg.GetStringSlice("single"))
	}

	if cfg.GetStringSlice("not_found") != nil {
		t.Errorf("GetStringSlice('not_found') should be nil")
	}
}

func TestMapConfig_GetSub(t *testing.T) {
	cfg := NewMapConfig(map[string]any{
		"app": map[string]any{
			"name": "myapp",
			"db": map[string]any{
				"host": "localhost",
			},
		},
		"not_map": "value",
	})

	sub, ok := cfg.GetSub("app")
	if !ok {
		t.Fatal("expected GetSub('app') ok = true")
	}
	if sub.Get("name") != "myapp" {
		t.Errorf("sub.Get('name') = %v, expected 'myapp'", sub.Get("name"))
	}

	db, ok := sub.GetSub("db")
	if !ok {
		t.Fatal("expected GetSub('db') ok = true")
	}
	if db.Get("host") != "localhost" {
		t.Errorf("db.Get('host') = %v, expected 'localhost'", db.Get("host"))
	}

	_, ok = cfg.GetSub("not_map")
	if ok {
		t.Error("expected GetSub('not_map') ok = false")
	}

	_, ok = cfg.GetSub("missing")
	if ok {
		t.Error("expected GetSub('missing') ok = false")
	}
}

func TestMapConfig_All(t *testing.T) {
	original := map[string]any{"key": "value"}
	cfg := NewMapConfig(original)
	copied := cfg.All()

	if !reflect.DeepEqual(copied, original) {
		t.Errorf("All() = %v, expected %v", copied, original)
	}

	copied["key"] = "modified"
	if original["key"] == "modified" {
		t.Error("All() should return a copy, not original")
	}
}
