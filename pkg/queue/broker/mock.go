package broker

import (
	"strings"

	"github.com/shuldan/framework/pkg/app"
	"github.com/shuldan/framework/pkg/contracts"
)

type mockContainer struct {
	instances map[string]interface{}
	factories map[string]func(contracts.DIContainer) (interface{}, error)
}

func newMockContainer() *mockContainer {
	return &mockContainer{
		instances: make(map[string]interface{}),
		factories: make(map[string]func(contracts.DIContainer) (interface{}, error)),
	}
}

func (m *mockContainer) Has(name string) bool {
	_, hasInstance := m.instances[name]
	_, hasFactory := m.factories[name]
	return hasInstance || hasFactory
}

func (m *mockContainer) Instance(name string, value interface{}) error {
	if _, exists := m.instances[name]; exists {
		return app.ErrDuplicateInstance
	}
	m.instances[name] = value
	return nil
}

func (m *mockContainer) Factory(name string, factory func(contracts.DIContainer) (interface{}, error)) error {
	if _, exists := m.factories[name]; exists {
		return app.ErrDuplicateFactory
	}
	m.factories[name] = factory
	return nil
}

func (m *mockContainer) Resolve(name string) (interface{}, error) {
	if instance, exists := m.instances[name]; exists {
		return instance, nil
	}

	if factory, exists := m.factories[name]; exists {
		return factory(m)
	}

	return nil, app.ErrValueNotFound.WithDetail("name", name)
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
