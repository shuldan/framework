package logger

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestLoggerMethods(t *testing.T) {
	buf := &bytes.Buffer{}
	logger, err := NewLogger(WithWriter(buf), WithText(), WithLevel(levelTrace))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		method func(string, ...any)
		level  slog.Level
		prefix string
	}{
		{logger.Trace, levelTrace, "TRACE"},
		{logger.Debug, slog.LevelDebug, "DEBUG"},
		{logger.Info, slog.LevelInfo, "INFO"},
		{logger.Warn, slog.LevelWarn, "WARN"},
		{logger.Error, slog.LevelError, "ERROR"},
		{logger.Critical, levelCritical, "CRITICAL"},
	}

	for _, tt := range tests {
		buf.Reset()
		t.Run(tt.prefix, func(t *testing.T) {
			tt.method("test", "key", "val")
			output := buf.String()
			if !strings.Contains(output, tt.prefix) || !strings.Contains(output, "key=\"val\"") {
				t.Errorf("Expected %q in output, got: %q", tt.prefix, output)
			}
		})
	}
}

func TestConvertArgs_OddArgs(t *testing.T) {
	buf := &bytes.Buffer{}
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, nil)))

	args := []any{"key1", "val1", "key2"}
	attrs := convertArgs(args)

	if len(attrs) != 2 {
		t.Errorf("Expected 2 attrs, got %d", len(attrs))
	}
	if attrs[1].Key != "MISSING_KEY" {
		t.Errorf("Expected MISSING_KEY, got %q", attrs[1].Key)
	}

	if !strings.Contains(buf.String(), "odd number of args") {
		t.Error("Expected odd args warning")
	}
}

func TestConvertArgs_NonStringKey(t *testing.T) {
	args := []any{42, "value"}
	attrs := convertArgs(args)
	if !strings.HasPrefix(attrs[0].Key, "NON_STRING_KEY_int") {
		t.Errorf("Expected NON_STRING_KEY_, got %q", attrs[0].Key)
	}
}

func TestLogger_With(t *testing.T) {
	buf := &bytes.Buffer{}
	logger, _ := NewLogger(WithWriter(buf), WithText())

	l2 := logger.With("service", "auth")
	l2.Info("login")

	output := buf.String()
	if !strings.Contains(output, "service=\"auth\"") {
		t.Error("With: expected attr not found")
	}
}

func TestNewLogger_DefaultOptions(t *testing.T) {
	t.Parallel()
	logger, err := NewLogger()
	if err != nil {
		t.Fatalf("NewLogger() failed: %v", err)
	}
	if logger == nil {
		t.Fatal("NewLogger() returned nil")
	}
}

func TestNewLogger_JSONMode(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	logger, err := NewLogger(WithJSON(), WithWriter(buf))
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}

	logger.Info("test", "key", "value")
	output := buf.String()
	if !strings.Contains(output, "{") || !strings.Contains(output, "}") {
		t.Error("Expected JSON output")
	}
}

func TestNewLogger_WithSourceOption(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	logger, err := NewLogger(WithSource(), WithJSON(), WithWriter(buf))
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}

	logger.Info("test")
	output := buf.String()
	if !strings.Contains(output, "source") {
		t.Error("Expected source info in output")
	}
}

func TestConvertArgs_EmptyArgs(t *testing.T) {
	t.Parallel()
	attrs := convertArgs([]any{})
	if len(attrs) != 0 {
		t.Errorf("Expected 0 attrs, got %d", len(attrs))
	}
}

func TestConvertArgs_ValidPairs(t *testing.T) {
	t.Parallel()
	args := []any{"key1", "val1", "key2", 42}
	attrs := convertArgs(args)

	if len(attrs) != 2 {
		t.Fatalf("Expected 2 attrs, got %d", len(attrs))
	}
	if attrs[0].Key != "key1" || attrs[0].Value.String() != "val1" {
		t.Errorf("First attr incorrect: %v", attrs[0])
	}
	if attrs[1].Key != "key2" || attrs[1].Value.String() != "42" {
		t.Errorf("Second attr incorrect: %v", attrs[1])
	}
}

func TestLogger_LevelFiltering(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	logger, _ := NewLogger(WithWriter(buf), WithLevel(slog.LevelWarn))

	logger.Debug("debug msg")
	logger.Info("info msg")
	output := buf.String()

	if strings.Contains(output, "debug msg") || strings.Contains(output, "info msg") {
		t.Error("Debug/Info should be filtered with Warn level")
	}

	buf.Reset()
	logger.Warn("warn msg")
	logger.Error("error msg")
	output = buf.String()

	if !strings.Contains(output, "warn msg") || !strings.Contains(output, "error msg") {
		t.Error("Warn/Error should pass with Warn level")
	}
}
