package config

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/shuldan/framework/pkg/contracts"
)

type MapConfig struct {
	values map[string]any
}

var _ contracts.Config = (*MapConfig)(nil)

func (c *MapConfig) Has(key string) bool {
	_, ok := c.find(key)
	return ok
}

func (c *MapConfig) Get(key string) any {
	value, _ := c.find(key)
	return value
}

func (c *MapConfig) GetString(key string, defaultVal ...string) string {
	v, ok := c.find(key)
	if !ok {
		return getFirst(defaultVal)
	}
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func (c *MapConfig) GetInt(key string, defaultVal ...int) int {
	v, ok := c.find(key)
	if !ok {
		return getFirst(defaultVal)
	}
	if i, ok := v.(int); ok {
		return i
	}
	if i, ok := v.(int64); ok {
		if i < int64(math.MinInt) || i > int64(math.MaxInt) {
			return getFirst(defaultVal)
		}
		return int(i)
	}
	if i, ok := v.(uint64); ok {
		if i > uint64(math.MaxInt) {
			return getFirst(defaultVal)
		}
		return int(i)
	}
	if f, ok := v.(float64); ok {
		if f < float64(math.MinInt) || f > float64(math.MaxInt) {
			return getFirst(defaultVal)
		}
		return int(f)
	}
	if b, ok := v.(bool); ok {
		if b {
			return 1
		}
		return 0
	}
	if s, ok := v.(string); ok {
		if i, err := strconv.Atoi(s); err == nil {
			return i
		}
	}
	return getFirst(defaultVal)
}

func (c *MapConfig) GetInt64(key string, defaultVal ...int64) int64 {
	v, ok := c.find(key)
	if !ok {
		return getFirst(defaultVal)
	}
	if i, ok := v.(int64); ok {
		return i
	}
	if i, ok := v.(int); ok {
		return int64(i)
	}
	if i, ok := v.(uint64); ok {
		if i > math.MaxInt64 {
			return getFirst(defaultVal)
		}
		return int64(i)
	}
	if f, ok := v.(float64); ok {
		if f < float64(math.MinInt64) || f > float64(math.MaxInt64) {
			return getFirst(defaultVal)
		}
		return int64(f)
	}
	if b, ok := v.(bool); ok {
		return map[bool]int64{true: 1, false: 0}[b]
	}
	if s, ok := v.(string); ok {
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i
		}
	}
	return getFirst(defaultVal)
}

func (c *MapConfig) GetFloat64(key string, defaultVal ...float64) float64 {
	v, ok := c.find(key)
	if !ok {
		return getFirst(defaultVal)
	}
	if f, ok := v.(float64); ok {
		return f
	}
	if i, ok := v.(int); ok {
		return float64(i)
	}
	if i, ok := v.(int64); ok {
		return float64(i)
	}
	if s, ok := v.(string); ok {
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f
		}
	}
	return getFirst(defaultVal)
}

func (c *MapConfig) GetBool(key string, defaultVal ...bool) bool {
	v, ok := c.find(key)
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

func (c *MapConfig) GetStringSlice(key string, separator ...string) []string {
	v, ok := c.find(key)
	if !ok {
		return nil
	}
	if v == nil {
		return nil
	}

	sep := ","
	if len(separator) > 0 {
		sep = separator[0]
	}

	switch val := v.(type) {
	case []string:
		return val
	case []any:
		result := make([]string, len(val))
		for i, item := range val {
			result[i] = fmt.Sprintf("%v", item)
		}
		return result
	case string:
		parts := strings.Split(val, sep)
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		return parts
	default:
		return []string{fmt.Sprintf("%v", v)}
	}
}

func (c *MapConfig) GetSub(key string) (contracts.Config, bool) {
	sub, ok := c.find(key)
	if !ok {
		return nil, false
	}
	if subMap, ok := sub.(map[string]any); ok {
		return NewMapConfig(subMap), true
	}
	return nil, false
}

func (c *MapConfig) All() map[string]any {
	return cloneMap(c.values)
}

func (c *MapConfig) find(path string) (any, bool) {
	keys := strings.Split(path, ".")
	var current any = c.values

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

func cloneMap(m map[string]any) map[string]any {
	cp := make(map[string]any, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}
