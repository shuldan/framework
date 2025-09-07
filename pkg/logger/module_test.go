package logger

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/shuldan/framework/pkg/contracts"
)

type mockContainer struct {
	items     map[string]interface{}
	factories map[string]func(container contracts.DIContainer) (interface{}, error)
}

func (m *mockContainer) Has(name string) bool {
	if _, ok := m.items[name]; ok {
		return true
	}
	_, ok := m.factories[name]
	return ok
}

func (m *mockContainer) Instance(name string, value interface{}) error {
	if name == "" {
		return io.EOF
	}
	m.items[name] = value
	return nil
}

func (m *mockContainer) Factory(name string, factory func(container contracts.DIContainer) (interface{}, error)) error {
	if name == "" {
		return io.EOF
	}
	m.factories[name] = factory

	if result, err := factory(m); err == nil {
		m.items[name] = result
	}
	return nil
}

func (m *mockContainer) Resolve(name string) (interface{}, error) {
	if val, ok := m.items[name]; ok {
		return val, nil
	}

	if factory, ok := m.factories[name]; ok {
		result, err := factory(m)
		if err != nil {
			return nil, err
		}
		m.items[name] = result
		return result, nil
	}

	return nil, io.EOF
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
	container := &mockContainer{
		items:     make(map[string]interface{}),
		factories: make(map[string]func(container contracts.DIContainer) (interface{}, error)),
	}

	err := m.Register(container)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if _, ok := container.items["logger"]; !ok {
		t.Error("Logger not registered in container")
	}
}

func TestModule_Start(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	logger, _ := NewLogger(WithWriter(buf))

	container := &mockContainer{
		items:     map[string]interface{}{"logger": logger},
		factories: make(map[string]func(container contracts.DIContainer) (interface{}, error)),
	}
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

	container := &mockContainer{
		items:     map[string]interface{}{"logger": logger},
		factories: make(map[string]func(container contracts.DIContainer) (interface{}, error)),
	}
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
			"level":      "debug",
			"format":     "json",
			"add_source": true,
			"color":      false,
		},
	}

	options := m.optionsFromFileConfig(cfg)

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
	container := &mockContainer{
		items:     make(map[string]interface{}),
		factories: make(map[string]func(container contracts.DIContainer) (interface{}, error)),
	}

	options := m.getLoggerOptions(container)

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
