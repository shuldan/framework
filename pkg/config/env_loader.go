package config

import (
	"os"
	"strconv"
	"strings"
)

type EnvConfigLoader struct {
	prefix string
}

func (l *EnvConfigLoader) Load() (map[string]any, error) {
	config := make(map[string]any)

	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, l.prefix) {
			continue
		}

		parts := strings.SplitN(env, "=", 2)
		key := parts[0]
		value := parts[1]

		configKey := strings.ToLower(strings.TrimPrefix(key, l.prefix))

		configKey = strings.ReplaceAll(configKey, "__", ".")

		var typedValue any = value
		if b, err := strconv.ParseBool(value); err == nil {
			typedValue = b
		} else if i, err := strconv.Atoi(value); err == nil {
			typedValue = i
		} else if f, err := strconv.ParseFloat(value, 64); err == nil {
			typedValue = f
		}

		setNested(config, configKey, typedValue)
	}

	return config, nil
}

func setNested(m map[string]any, key string, value any) {
	keys := strings.Split(key, ".")
	last := len(keys) - 1

	current := m
	for i, k := range keys {
		if i == last {
			current[k] = value
		} else {
			if _, ok := current[k]; !ok {
				current[k] = make(map[string]any)
			}
			if next, ok := current[k].(map[string]any); ok {
				current = next
			} else {

				current[k] = make(map[string]any)
				current = current[k].(map[string]any)
			}
		}
	}
}
