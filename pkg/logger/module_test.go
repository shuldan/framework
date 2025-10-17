package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/shuldan/framework/pkg/app"
	"github.com/shuldan/framework/pkg/contracts"
)

type mockContainer struct {
	mu        sync.RWMutex
	instances map[reflect.Type]interface{}
	factories map[reflect.Type]func(contracts.DIContainer) (interface{}, error)
}

func newMockContainer() *mockContainer {
	return &mockContainer{
		instances: make(map[reflect.Type]interface{}),
		factories: make(map[reflect.Type]func(contracts.DIContainer) (interface{}, error)),
	}
}

func (m *mockContainer) Has(abstract reflect.Type) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, hasInstance := m.instances[abstract]
	_, hasFactory := m.factories[abstract]
	return hasInstance || hasFactory
}

func (m *mockContainer) Instance(abstract reflect.Type, concrete interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if _, exists := m.instances[abstract]; exists {
		return app.ErrDuplicateInstance.WithDetail("type", abstract.String())
	}
	m.instances[abstract] = concrete
	return nil
}

func (m *mockContainer) Factory(abstract reflect.Type, factory func(contracts.DIContainer) (interface{}, error)) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if _, exists := m.factories[abstract]; exists {
		return app.ErrDuplicateFactory.WithDetail("type", abstract.String())
	}
	m.factories[abstract] = factory
	return nil
}

func (m *mockContainer) Resolve(abstract reflect.Type) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if instance, exists := m.instances[abstract]; exists {
		return instance, nil
	}

	if factory, exists := m.factories[abstract]; exists {
		return factory(m)
	}

	return nil, app.ErrValueNotFound.WithDetail("type", abstract.String())
}

type mockConfig struct {
	data map[string]interface{}
}

func (m *mockConfig) Has(key string) bool {
	_, ok := m.data[key]
	return ok
}

func (m *mockConfig) Get(key string) any {
	return m.data[key]
}

func (m *mockConfig) GetString(key string, defaultVal ...string) string {
	if val, ok := m.data[key].(string); ok {
		return val
	}
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return ""
}

func (m *mockConfig) GetInt(key string, defaultVal ...int) int {
	if val, ok := m.data[key].(int); ok {
		return val
	}
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return 0
}

func (m *mockConfig) GetInt64(key string, defaultVal ...int64) int64 {
	if val, ok := m.data[key].(int64); ok {
		return val
	}
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return 0
}

func (m *mockConfig) GetFloat64(key string, defaultVal ...float64) float64 {
	if val, ok := m.data[key].(float64); ok {
		return val
	}
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return 0.0
}

func (m *mockConfig) GetBool(key string, defaultVal ...bool) bool {
	if val, ok := m.data[key].(bool); ok {
		return val
	}
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return false
}

func (m *mockConfig) GetStringSlice(key string, separator ...string) []string {
	if val, ok := m.data[key].([]string); ok {
		return val
	}
	if val, ok := m.data[key].(string); ok {
		sep := ","
		if len(separator) > 0 {
			sep = separator[0]
		}
		return strings.Split(val, sep)
	}
	return []string{}
}

func (m *mockConfig) GetSub(key string) (contracts.Config, bool) {
	if key == "logger" {
		return m, true
	}
	if val, ok := m.data[key]; ok {
		if cfg, ok := val.(contracts.Config); ok {
			return cfg, true
		}
	}
	return nil, false
}

func (m *mockConfig) All() map[string]any {
	result := make(map[string]any)
	for k, v := range m.data {
		result[k] = v
	}
	return result
}

type mockAppContext struct {
	ctx       context.Context
	container contracts.DIContainer
	startTime time.Time
	stopTime  time.Time
	running   bool
}

func (m *mockAppContext) AppRegistry() contracts.AppRegistry {
	return nil
}

func (m *mockAppContext) Ctx() context.Context             { return m.ctx }
func (m *mockAppContext) Container() contracts.DIContainer { return m.container }
func (m *mockAppContext) AppName() string                  { return "testapp" }
func (m *mockAppContext) Version() string                  { return "1.0.0" }
func (m *mockAppContext) Environment() string              { return "test" }
func (m *mockAppContext) StartTime() time.Time             { return m.startTime }
func (m *mockAppContext) StopTime() time.Time              { return m.stopTime }
func (m *mockAppContext) IsRunning() bool                  { return m.running }
func (m *mockAppContext) Stop() {
	m.running = false
	m.stopTime = time.Now()
}

func TestModule_Name(t *testing.T) {
	t.Parallel()
	m := NewModule()
	if m.Name() != "logger" {
		t.Errorf("Expected name 'logger', got %q", m.Name())
	}
}

func TestModule_Register(t *testing.T) {
	t.Parallel()
	m := NewModule()
	container := newMockContainer()

	err := m.Register(container)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if _, ok := container.factories[reflect.TypeOf((*contracts.Logger)(nil)).Elem()]; !ok {
		t.Error("Logger not registered in container")
	}
}

func TestModule_Start(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	logger, _ := NewLogger(WithWriter(buf))

	container := newMockContainer()
	container.instances[reflect.TypeOf((*contracts.Logger)(nil)).Elem()] = logger

	ctx := &mockAppContext{
		container: container,
		startTime: time.Now(),
	}

	m := NewModule()
	err := m.(*module).Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Logging started") {
		t.Error("Expected 'Logging started' message")
	}
	if !strings.Contains(output, "app=\"testapp\"") {
		t.Error("Expected app name in output")
	}
}

func TestModule_Stop(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	logger, _ := NewLogger(WithWriter(buf))

	container := newMockContainer()
	container.instances[reflect.TypeOf((*contracts.Logger)(nil)).Elem()] = logger

	ctx := &mockAppContext{
		container: container,
		startTime: time.Now().Add(-time.Hour),
	}

	m := NewModule()
	err := m.(*module).Stop(ctx)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Logging stopped") {
		t.Error("Expected 'Logging stopped' message")
	}
}

func TestModule_ParseLevel(t *testing.T) {
	t.Parallel()
	m := &module{}
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"trace", levelTrace},
		{"TRACE", levelTrace},
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{"critical", levelCritical},
		{"fatal", levelCritical},
		{"unknown", slog.LevelInfo},
		{"", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level := m.parseLevel(tt.input)
			if level != tt.expected {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.input, level, tt.expected)
			}
		})
	}
}

func TestModule_OptionsFromFileConfig(t *testing.T) {
	t.Parallel()
	m := &module{}
	cfg := &mockConfig{
		data: map[string]interface{}{
			"level":          "debug",
			"format":         "json",
			"include_caller": true,
			"color":          false,
		},
	}

	options, _ := m.optionsFromFileConfig(cfg)

	testCfg := &config{}
	for _, opt := range options {
		opt(testCfg)
	}

	if testCfg.level != slog.LevelDebug {
		t.Errorf("Expected debug level, got %v", testCfg.level)
	}
	if !testCfg.json {
		t.Error("Expected JSON format")
	}
	if !testCfg.addSource {
		t.Error("Expected add_source to be true")
	}
	if testCfg.wantColor {
		t.Error("Expected color to be false")
	}
}

func TestModule_GetLoggerOptions_NoConfig(t *testing.T) {
	t.Parallel()
	m := &module{}
	container := newMockContainer()

	options, _ := m.getLoggerOptions(container)

	testCfg := &config{}
	for _, opt := range options {
		opt(testCfg)
	}

	if testCfg.level != slog.LevelInfo {
		t.Errorf("Default level should be Info, got %v", testCfg.level)
	}
	if testCfg.json {
		t.Error("Default format should be text")
	}
}

func TestModule_ConfigurationOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		config         map[string]interface{}
		validateOutput func(t *testing.T, output string)
	}{
		{
			name: "json_format_with_error_level",
			config: map[string]interface{}{
				"level":          "error",
				"format":         "json",
				"output":         "stdout",
				"include_caller": false,
				"enable_colors":  false,
			},
			validateOutput: func(t *testing.T, output string) {
				if strings.Contains(output, "debug message") {
					t.Error("Debug messages should be filtered")
				}
				if strings.Contains(output, "info message") {
					t.Error("Info messages should be filtered")
				}
				if !strings.Contains(output, "error message") {
					t.Error("Error message should be logged")
				}
				if !strings.Contains(output, `{`) || !strings.Contains(output, `}`) {
					t.Error("Expected JSON format")
				}
				if !strings.Contains(output, `"key":"value"`) {
					t.Error("Expected key-value in JSON")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			output := captureLoggerOutput(t, tt.config, func(logger contracts.Logger) {
				logger.Debug("debug message")
				logger.Info("info message")
				logger.Error("error message", "key", "value")
			})

			tt.validateOutput(t, string(output))
		})
	}
}

func TestModule_FileOutput(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join("test", "app.log")

	cfg := &mockConfig{
		data: map[string]interface{}{
			"level":     "info",
			"format":    "text",
			"output":    "file",
			"base_dir":  tempDir,
			"file_path": logFile,
		},
	}

	m := &module{}
	options, err := m.optionsFromFileConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to get options: %v", err)
	}

	logger, err := NewLogger(options...)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	logger.Info("test file logging", "key", "value")

	absLogFile := filepath.Join(tempDir, logFile)
	if _, err := os.Stat(absLogFile); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}

	content, err := os.ReadFile(absLogFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), "test file logging") {
		t.Error("Log message not found in file")
	}
}

func TestModule_ParseLevel_AllLevels(t *testing.T) {
	m := &module{}
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"trace", levelTrace},
		{"TRACE", levelTrace},
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"WARN", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"WARNING", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR", slog.LevelError},
		{"critical", levelCritical},
		{"CRITICAL", levelCritical},
		{"fatal", levelCritical},
		{"FATAL", levelCritical},
		{"unknown", slog.LevelInfo},
		{"", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level := m.parseLevel(tt.input)
			if level != tt.expected {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.input, level, tt.expected)
			}
		})
	}
}

func TestModule_GetWriter(t *testing.T) {
	m := &module{}

	tests := []struct {
		name     string
		output   string
		expected io.Writer
	}{
		{"stdout", "stdout", os.Stdout},
		{"stderr", "stderr", os.Stderr},
		{"STDOUT", "STDOUT", os.Stdout},
		{"unknown", "unknown", os.Stdout},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &mockConfig{data: map[string]interface{}{}}
			writer, err := m.getWriter(tt.output, cfg)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if writer != tt.expected {
				t.Errorf("Expected %v writer", tt.name)
			}
		})
	}
}

func TestModule_ColorConfiguration(t *testing.T) {
	tests := []struct {
		name         string
		format       string
		enableColors bool
		expectColors bool
	}{
		{"text_with_colors", "text", true, true},
		{"text_without_colors", "text", false, false},
		{"json_with_colors", "json", true, false},
		{"json_without_colors", "json", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &mockConfig{
				data: map[string]interface{}{
					"format":        tt.format,
					"enable_colors": tt.enableColors,
				},
			}

			m := &module{}
			options, _ := m.optionsFromFileConfig(cfg)

			config := &config{}
			for _, opt := range options {
				opt(config)
			}

			if tt.expectColors && !config.wantColor {
				t.Error("Expected colors to be enabled")
			} else if !tt.expectColors && config.wantColor {
				t.Error("Expected colors to be disabled")
			}
		})
	}
}

func (m *mockConfig) GetWriter() io.Writer {
	if w, ok := m.data["_writer"].(io.Writer); ok {
		return w
	}
	return nil
}

func TestModule_JSONFormat_WithCaller(t *testing.T) {
	configData := map[string]interface{}{
		"level":          "info",
		"format":         "json",
		"output":         "stdout",
		"include_caller": true,
	}
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	var logger contracts.Logger
	done := make(chan struct{})
	go func() {
		defer close(done)
		cfg := &mockConfig{data: configData}
		m := &module{}
		options, err := m.optionsFromFileConfig(cfg)
		if err != nil {
			t.Errorf("Failed to get options: %v", err)
			return
		}
		logger, err = NewLogger(options...)
		if err != nil {
			t.Errorf("Failed to create logger: %v", err)
			return
		}
		logger.Info("test with caller")
		_ = w.Close()
	}()

	buf := &bytes.Buffer{}
	_, _ = io.Copy(buf, r)
	os.Stdout = originalStdout
	<-done

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &logEntry); err != nil {
		t.Fatalf("Output is not valid JSON: %v\nRaw: %q", err, lines[0])
	}

	if _, ok := logEntry["source"]; !ok {
		t.Error("Expected 'source' field in JSON output when include_caller=true")
	}
	if file, ok := logEntry["source"].(map[string]interface{})["file"]; !ok || file == "" {
		t.Error("Source file should be present")
	}
}

func TestModule_FileOutput_JSONFormat(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join("logs", "app.log")

	configData := map[string]interface{}{
		"level":     "info",
		"format":    "json",
		"output":    "file",
		"base_dir":  tempDir,
		"file_path": logFile,
	}

	m := &module{}
	cfg := &mockConfig{data: configData}
	options, err := m.optionsFromFileConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to get options: %v", err)
	}

	logger, err := NewLogger(options...)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	logger.Info("test json to file", "user", "alice", "action", "login")

	absLogFile := filepath.Join(tempDir, logFile)
	content, err := os.ReadFile(absLogFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if len(content) == 0 {
		t.Fatal("Log file is empty")
	}

	var logEntry map[string]interface{}
	if err := json.Unmarshal(content, &logEntry); err != nil {
		t.Fatalf("Log file does not contain valid JSON: %v", err)
	}

	if msg, ok := logEntry["msg"].(string); !ok || msg != "test json to file" {
		t.Errorf("Expected msg 'test json to file', got %q", msg)
	}
	if _, ok := logEntry["user"]; !ok {
		t.Error("Expected 'user' attribute in JSON")
	}
}

func TestModule_InvalidLogLevel_DefaultsToInfo(t *testing.T) {
	configData := map[string]interface{}{
		"level": "bogus",
	}

	m := &module{}
	cfg := &mockConfig{data: configData}
	options, err := m.optionsFromFileConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to get options: %v", err)
	}

	testCfg := &config{}
	for _, opt := range options {
		opt(testCfg)
	}

	if testCfg.level != slog.LevelInfo {
		t.Errorf("Expected default level 'info' for invalid level, got %v", testCfg.level)
	}
}

func TestModule_UnicodeSupport_InJSON(t *testing.T) {
	configData := map[string]interface{}{
		"level":  "info",
		"format": "json",
		"output": "stdout",
	}

	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	var logger contracts.Logger
	done := make(chan struct{})
	go func() {
		defer close(done)
		cfg := &mockConfig{data: configData}
		m := &module{}
		options, err := m.optionsFromFileConfig(cfg)
		if err != nil {
			t.Errorf("Failed to get options: %v", err)
			return
		}
		logger, err = NewLogger(options...)
		if err != nil {
			t.Errorf("Failed to create logger: %v", err)
			return
		}
		logger.Info("Привет, мир!", "пользователь", "Иван")
		_ = w.Close()
	}()

	buf := &bytes.Buffer{}
	_, _ = io.Copy(buf, r)
	os.Stdout = originalStdout
	<-done

	output := buf.String()
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if msg, ok := logEntry["msg"].(string); !ok || msg != "Привет, мир!" {
		t.Errorf("Expected Unicode message, got %q", msg)
	}
	if user, ok := logEntry["пользователь"].(string); !ok || user != "Иван" {
		t.Errorf("Expected Cyrillic key/value, got %q", user)
	}
}

func TestModule_ReplaceAttr_DoesNotBreakDefaults(t *testing.T) {
	configData := map[string]interface{}{
		"level":  "warn",
		"format": "text",
		"output": "stdout",
	}

	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	var logger contracts.Logger
	done := make(chan struct{})
	go func() {
		defer close(done)
		cfg := &mockConfig{data: configData}
		m := &module{}
		options, err := m.optionsFromFileConfig(cfg)
		if err != nil {
			t.Errorf("Failed to get options: %v", err)
			return
		}
		logger, err = NewLogger(options...)
		if err != nil {
			t.Errorf("Failed to create logger: %v", err)
			return
		}
		logger.Info("should be filtered")
		logger.Warn("warning msg")
		_ = w.Close()
	}()

	buf := &bytes.Buffer{}
	_, _ = io.Copy(buf, r)
	os.Stdout = originalStdout
	<-done

	output := buf.String()

	if strings.Contains(output, "INFO") || strings.Contains(output, "should be filtered") {
		t.Error("Info message should be filtered at warn level")
	}
	if !strings.Contains(output, "WARN") || !strings.Contains(output, "warning msg") {
		t.Error("Warn message should be logged")
	}
}
