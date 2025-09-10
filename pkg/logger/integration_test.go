package logger

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/shuldan/framework/pkg/contracts"
)

func TestLoggerIntegration_JSON_CompleteFields(t *testing.T) {
	t.Parallel()

	configData := map[string]interface{}{
		"level":          "warn",
		"format":         "json",
		"output":         "stdout",
		"include_caller": true,
		"enable_colors":  false,
	}

	output := captureLoggerOutput(t, configData, func(logger contracts.Logger) {
		logger.Warn("warn msg", "user", "john", "action", "login")
	})

	var logEntry map[string]interface{}
	if err := json.Unmarshal(output, &logEntry); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// Проверяем обязательные поля
	assertHasKeys(t, logEntry, "time", "level", "msg", "user", "action")
	assertValue(t, logEntry, "msg", "warn msg")
	assertValue(t, logEntry, "user", "john")
	assertValue(t, logEntry, "action", "login")
	assertValue(t, logEntry, "level", "WARN")

	// Проверяем source
	if source, ok := logEntry["source"].(map[string]interface{}); ok {
		if _, ok := source["file"]; !ok {
			t.Error("Expected 'source.file' in JSON output")
		}
		if _, ok := source["line"]; !ok {
			t.Error("Expected 'source.line' in JSON output")
		}
	} else {
		t.Error("Expected 'source' object in JSON output")
	}
}

func TestLoggerIntegration_JSON_LevelFiltering(t *testing.T) {
	t.Parallel()

	configData := map[string]interface{}{
		"level":  "warn",
		"format": "json",
		"output": "stdout",
	}

	output := captureLoggerOutput(t, configData, func(logger contracts.Logger) {
		logger.Info("should be filtered")
		logger.Warn("warn msg")
	})

	lines := splitLines(string(output))
	if len(lines) != 1 {
		t.Fatalf("Expected 1 log line (only warn+), got %d", len(lines))
	}
	if strings.Contains(string(output), "should be filtered") {
		t.Error("Info message should be filtered")
	}
	if !strings.Contains(string(output), "warn msg") {
		t.Error("Warn message should be logged")
	}
}

func TestLoggerIntegration_JSON_CriticalLevel(t *testing.T) {
	t.Parallel()

	configData := map[string]interface{}{
		"level":  "debug",
		"format": "json",
		"output": "stdout",
	}

	output := captureLoggerOutput(t, configData, func(logger contracts.Logger) {
		logger.Critical("server is down")
	})

	var logEntry map[string]interface{}
	if err := json.Unmarshal(output, &logEntry); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	assertValue(t, logEntry, "level", "CRITICAL")
	assertValue(t, logEntry, "msg", "server is down")
}

func captureLoggerOutput(t *testing.T, configData map[string]interface{}, logFunc func(logger contracts.Logger)) []byte {
	t.Helper()

	var buf bytes.Buffer

	cfg := &mockConfig{data: configData}
	m := &module{}
	options, err := m.optionsFromFileConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to get options: %v", err)
	}

	options = append(options, WithWriter(&buf))
	logger, err := NewLogger(options...)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	logFunc(logger)

	return buf.Bytes()
}

func splitLines(output string) []string {
	output = strings.TrimSpace(output)
	if output == "" {
		return []string{}
	}
	return strings.Split(output, "\n")
}

func assertHasKeys(t *testing.T, m map[string]interface{}, keys ...string) {
	t.Helper()
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			t.Errorf("Expected key %q in log entry", k)
		}
	}
}

func assertValue(t *testing.T, m map[string]interface{}, key string, expected interface{}) {
	t.Helper()
	if val, ok := m[key]; !ok {
		t.Errorf("Missing key %q", key)
	} else if val != expected {
		t.Errorf("Key %q: got %v, want %v", key, val, expected)
	}
}

func TestLoggerIntegration_WithContext(t *testing.T) {
	buf := &bytes.Buffer{}
	logger, _ := NewLogger(
		WithWriter(buf),
		WithLevel(slog.LevelDebug),
		WithText(),
	)
	contextLogger := logger.With("request_id", "12345", "service", "api")
	contextLogger.Info("processing request", "method", "GET", "path", "/users")
	contextLogger.Error("request failed", "status", 500)
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 log lines, got %d", len(lines))
	}
	for i, line := range lines {
		if !strings.Contains(line, `request_id="12345"`) {
			t.Errorf("Line %d missing request_id in context: %s", i, line)
		}
		if !strings.Contains(line, `service="api"`) {
			t.Errorf("Line %d missing service in context: %s", i, line)
		}
	}
}

func TestLoggerIntegration_TextFormat(t *testing.T) {
	t.Parallel()

	configData := map[string]interface{}{
		"level":          "debug",
		"format":         "text",
		"output":         "stdout",
		"include_caller": false,
		"enable_colors":  false,
	}

	output := captureLoggerOutput(t, configData, func(logger contracts.Logger) {
		logger.Info("test message", "user", "john", "action", "login")
	})

	outputStr := string(output)
	if !strings.Contains(outputStr, "INFO") {
		t.Error("Expected level INFO in output")
	}
	if !strings.Contains(outputStr, "test message") {
		t.Error("Expected message in output")
	}
	if !strings.Contains(outputStr, `user="john"`) {
		t.Error("Expected user attribute")
	}
	if !strings.Contains(outputStr, `action="login"`) {
		t.Error("Expected action attribute")
	}
}
