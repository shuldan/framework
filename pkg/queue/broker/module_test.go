package broker

import (
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/shuldan/framework/pkg/app"
	"github.com/shuldan/framework/pkg/contracts"
	"github.com/shuldan/framework/pkg/errors"
)

func TestModule_Register_Success_MemoryDriver(t *testing.T) {
	container := newMockContainer()
	module := NewModule()

	config := &mockConfig{
		data: map[string]interface{}{
			"queue": map[string]interface{}{
				"driver": "memory",
			},
		},
	}
	err := container.Instance(reflect.TypeOf((*contracts.Config)(nil)).Elem(), config)
	if err != nil {
		t.Fatalf("Failed to register config: %v", err)
	}

	logger := &mockLogger{}
	err = container.Instance(reflect.TypeOf((*contracts.Logger)(nil)).Elem(), logger)
	if err != nil {
		t.Fatalf("Failed to register logger: %v", err)
	}

	err = module.Register(container)
	if err != nil {
		t.Fatalf("Module registration failed: %v", err)
	}

	broker, err := container.Resolve(reflect.TypeOf((*contracts.Broker)(nil)).Elem())
	if err != nil {
		t.Fatalf("Failed to resolve queue broker: %v", err)
	}

	if broker == nil {
		t.Fatal("Queue broker should not be nil")
	}
}

func TestModule_Register_Success_RedisDriver(t *testing.T) {
	container := newMockContainer()
	module := NewModule()

	config := &mockConfig{
		data: map[string]interface{}{
			"queue": map[string]interface{}{
				"driver": "redis",
				"drivers": map[string]interface{}{
					"redis": map[string]interface{}{
						"prefix":               "testapp",
						"consumer_group":       "testgroup",
						"processing_timeout":   int64(30),
						"claim_interval":       int64(1),
						"max_claim_batch":      10,
						"block_timeout":        int64(500),
						"max_stream_length":    int64(10000),
						"approximate_trimming": true,
						"enable_claim":         true,
						"consumer_prefix":      "testnode",
						"client": map[string]interface{}{
							"address":  "localhost:6379",
							"username": "testuser",
							"password": "testpass",
						},
					},
				},
			},
		},
	}
	err := container.Instance(reflect.TypeOf((*contracts.Config)(nil)).Elem(), config)
	if err != nil {
		t.Fatalf("Failed to register config: %v", err)
	}

	logger := &mockLogger{}
	err = container.Instance(reflect.TypeOf((*contracts.Logger)(nil)).Elem(), logger)
	if err != nil {
		t.Fatalf("Failed to register logger: %v", err)
	}

	err = module.Register(container)
	if err != nil {
		t.Fatalf("Module registration failed: %v", err)
	}

	broker, err := container.Resolve(reflect.TypeOf((*contracts.Broker)(nil)).Elem())
	if err != nil {
		t.Fatalf("Failed to resolve queue broker: %v", err)
	}

	if broker == nil {
		t.Fatal("Queue broker should not be nil")
	}
}

func TestModule_Register_MissingConfig(t *testing.T) {
	container := newMockContainer()
	module := NewModule()

	logger := &mockLogger{}

	if err := container.Instance(reflect.TypeOf((*contracts.Logger)(nil)).Elem(), logger); err != nil {
		t.Fatalf("Failed to register logger: %v", err)
	}

	config := &mockConfig{}

	if err := container.Instance(reflect.TypeOf((*contracts.Config)(nil)).Elem(), config); err != nil {
		t.Fatalf("Failed to register config: %v", err)
	}

	err := module.Register(container)
	if err == nil {
		t.Fatal("Expected error when config is missing, got nil")
	}

	if !errors.Is(err, ErrQueueBrokerConfigNotFound) {
		t.Errorf("Expected ErrQueueBrokerConfigNotFound, got %v", err)
	}
}

func TestModule_Register_InvalidConfigInstance(t *testing.T) {
	container := newMockContainer()
	module := NewModule()

	err := container.Instance(reflect.TypeOf((*contracts.Config)(nil)).Elem(), "invalid config")
	if err != nil {
		t.Fatalf("Failed to register invalid config: %v", err)
	}

	logger := &mockLogger{}
	err = container.Instance(reflect.TypeOf((*contracts.Logger)(nil)).Elem(), logger)
	if err != nil {
		t.Fatalf("Failed to register logger: %v", err)
	}

	err = module.Register(container)
	if err == nil {
		t.Fatal("Expected error for invalid config instance, got nil")
	}

	if !errors.Is(err, ErrInvalidConfigInstance) {
		t.Errorf("Expected ErrInvalidConfigInstance, got %v", err)
	}
}

func TestModule_Register_InvalidLoggerInstance(t *testing.T) {
	container := newMockContainer()
	module := NewModule()

	config := &mockConfig{
		data: map[string]interface{}{
			"queue": map[string]interface{}{
				"driver": "memory",
			},
		},
	}
	err := container.Instance(reflect.TypeOf((*contracts.Config)(nil)).Elem(), config)
	if err != nil {
		t.Fatalf("Failed to register config: %v", err)
	}

	err = container.Instance(reflect.TypeOf((*contracts.Logger)(nil)).Elem(), "invalid logger")
	if err != nil {
		t.Fatalf("Failed to register invalid logger: %v", err)
	}

	err = module.Register(container)
	if err == nil {
		t.Fatal("Expected error for invalid logger instance, got nil")
	}

	if !errors.Is(err, ErrInvalidLoggerInstance) {
		t.Errorf("Expected ErrInvalidLoggerInstance, got %v", err)
	}
}

func TestModule_Register_MissingQueueConfig(t *testing.T) {
	container := newMockContainer()
	module := NewModule()

	config := &mockConfig{
		data: map[string]interface{}{
			"app": map[string]interface{}{
				"name": "testapp",
			},
		},
	}
	err := container.Instance(reflect.TypeOf((*contracts.Config)(nil)).Elem(), config)
	if err != nil {
		t.Fatalf("Failed to register config: %v", err)
	}

	logger := &mockLogger{}
	err = container.Instance(reflect.TypeOf((*contracts.Logger)(nil)).Elem(), logger)
	if err != nil {
		t.Fatalf("Failed to register logger: %v", err)
	}

	err = module.Register(container)
	if err == nil {
		t.Fatal("Expected error when queue config is missing, got nil")
	}

	if !errors.Is(err, ErrQueueBrokerConfigNotFound) {
		t.Errorf("Expected ErrQueueBrokerConfigNotFound, got %v", err)
	}
}

func TestModule_Register_UnsupportedDriver(t *testing.T) {
	container := newMockContainer()
	module := NewModule()

	config := &mockConfig{
		data: map[string]interface{}{
			"queue": map[string]interface{}{
				"driver": "unsupported",
			},
		},
	}
	err := container.Instance(reflect.TypeOf((*contracts.Config)(nil)).Elem(), config)
	if err != nil {
		t.Fatalf("Failed to register config: %v", err)
	}

	logger := &mockLogger{}
	err = container.Instance(reflect.TypeOf((*contracts.Logger)(nil)).Elem(), logger)
	if err != nil {
		t.Fatalf("Failed to register logger: %v", err)
	}

	err = module.Register(container)
	if err == nil {
		t.Fatal("Expected error for unsupported driver, got nil")
	}

	if !errors.Is(err, ErrUnsupportedQueueDriver) {
		t.Errorf("Expected ErrUnsupportedQueueDriver, got %v", err)
	}
}

func TestModule_Register_RedisDriver_MissingRedisConfig(t *testing.T) {
	container := newMockContainer()
	module := NewModule()

	config := &mockConfig{
		data: map[string]interface{}{
			"queue": map[string]interface{}{
				"driver": "redis",
			},
		},
	}
	err := container.Instance(reflect.TypeOf((*contracts.Config)(nil)).Elem(), config)
	if err != nil {
		t.Fatalf("Failed to register config: %v", err)
	}

	logger := &mockLogger{}
	err = container.Instance(reflect.TypeOf((*contracts.Logger)(nil)).Elem(), logger)
	if err != nil {
		t.Fatalf("Failed to register logger: %v", err)
	}

	err = module.Register(container)
	if err == nil {
		t.Fatal("Expected error when redis config is missing, got nil")
	}

	if !errors.Is(err, ErrRedisConfigNotFound) {
		t.Errorf("Expected ErrRedisConfigNotFound, got %v", err)
	}
}

func TestModule_Register_RedisDriver_MissingClientConfig(t *testing.T) {
	container := newMockContainer()
	module := NewModule()

	config := &mockConfig{
		data: map[string]interface{}{
			"queue": map[string]interface{}{
				"driver": "redis",
				"drivers": map[string]interface{}{
					"redis": map[string]interface{}{
						"prefix": "testapp",
					},
				},
			},
		},
	}
	err := container.Instance(reflect.TypeOf((*contracts.Config)(nil)).Elem(), config)
	if err != nil {
		t.Fatalf("Failed to register config: %v", err)
	}

	logger := &mockLogger{}
	err = container.Instance(reflect.TypeOf((*contracts.Logger)(nil)).Elem(), logger)
	if err != nil {
		t.Fatalf("Failed to register logger: %v", err)
	}

	err = module.Register(container)
	if err == nil {
		t.Fatal("Expected error when redis client config is missing, got nil")
	}

	if !errors.Is(err, ErrRedisClientNotConfigured) {
		t.Errorf("Expected ErrRedisClientNotConfigured, got %v", err)
	}
}

func TestModule_Start(t *testing.T) {
	module := NewModule()
	err := module.Start(nil)
	if err != nil {
		t.Errorf("Start should return nil, got %v", err)
	}
}

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
	if v, ok := m.data[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return ""
}

func (m *mockConfig) GetInt(key string, defaultVal ...int) int {
	if v, ok := m.data[key]; ok {
		if i, ok := v.(int); ok {
			return i
		}
	}
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return 0
}

func (m *mockConfig) GetInt64(key string, defaultVal ...int64) int64 {
	if v, ok := m.data[key]; ok {
		if i, ok := v.(int64); ok {
			return i
		}
		if i, ok := v.(int); ok {
			return int64(i)
		}
	}
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return 0
}

func (m *mockConfig) GetFloat64(key string, defaultVal ...float64) float64 {
	if v, ok := m.data[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
		if i, ok := v.(int); ok {
			return float64(i)
		}
	}
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return 0
}

func (m *mockConfig) GetBool(key string, defaultVal ...bool) bool {
	if v, ok := m.data[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return false
}

func (m *mockConfig) GetStringSlice(key string, separator ...string) []string {
	if v, ok := m.data[key]; ok {
		if s, ok := v.([]string); ok {
			return s
		}
	}
	return nil
}

func (m *mockConfig) GetSub(key string) (contracts.Config, bool) {
	sub, ok := m.find(key)
	if !ok {
		return nil, false
	}
	if subMap, ok := sub.(map[string]any); ok {
		return &mockConfig{data: subMap}, true
	}
	return nil, false
}

func (m *mockConfig) All() map[string]any {
	return m.data
}

func (m *mockConfig) find(path string) (any, bool) {
	keys := strings.Split(path, ".")
	var current any = m.data

	for _, k := range keys {
		if current == nil {
			return nil, false
		}

		switch cur := current.(type) {
		case map[string]any:
			next, exists := cur[k]
			if !exists {
				return nil, false
			}
			current = next
		case map[any]any:
			next, exists := cur[k]
			if !exists {
				return nil, false
			}
			current = next
		default:
			return nil, false
		}
	}

	return current, true
}

type mockLogger struct {
	logs []logEntry
}

type logEntry struct {
	level  string
	msg    string
	fields []interface{}
}

func (m *mockLogger) Debug(msg string, fields ...interface{}) {
	m.logs = append(m.logs, logEntry{"debug", msg, fields})
}

func (m *mockLogger) Info(msg string, fields ...interface{}) {
	m.logs = append(m.logs, logEntry{"info", msg, fields})
}

func (m *mockLogger) Warn(msg string, fields ...interface{}) {
	m.logs = append(m.logs, logEntry{"warn", msg, fields})
}

func (m *mockLogger) Error(msg string, fields ...interface{}) {
	m.logs = append(m.logs, logEntry{"error", msg, fields})
}

func (m *mockLogger) Critical(msg string, fields ...interface{}) {
	m.logs = append(m.logs, logEntry{"critical", msg, fields})
}

func (m *mockLogger) Trace(msg string, args ...any) {
	m.logs = append(m.logs, logEntry{"trace", msg, args})
}

func (m *mockLogger) With(_ ...any) contracts.Logger {
	return m
}
