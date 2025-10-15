package http

import (
	"strings"

	"github.com/shuldan/framework/pkg/contracts"
)

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

func (m *mockConfig) GetInt64(_ string, _ ...int64) int64 {
	panic("not implemented for mock")
}

func (m *mockConfig) GetFloat64(_ string, _ ...float64) float64 {
	panic("not implemented for mock")
}

func (m *mockConfig) GetBool(key string, defaultVal ...bool) bool {
	v, ok := m.find(key)
	if !ok {
		return getFirst(defaultVal)
	}
	if b, ok := v.(bool); ok {
		return b
	}
	if s, ok := v.(string); ok {
		switch strings.ToLower(s) {
		case "true", "1", "on", "yes", "y":
			return true
		case "false", "0", "off", "no", "n":
			return false
		}
	}
	if f, ok := v.(float64); ok {
		return f != 0
	}
	if i, ok := v.(int); ok {
		return i != 0
	}
	return getFirst(defaultVal)
}

func (m *mockConfig) GetStringSlice(_ string, _ ...string) []string {
	panic("not implemented for mock")
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

func getFirst[T any](values []T) T {
	var zero T
	if len(values) > 0 {
		return values[0]
	}
	return zero
}
