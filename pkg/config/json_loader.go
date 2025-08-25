package config

import (
	"encoding/json"
	"os"
)

type JSONConfigLoader struct {
	paths []string
}

func (l *JSONConfigLoader) Load() (map[string]any, error) {
	for _, path := range l.paths {
		if !fileExists(path) {
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var config map[string]any
		if err = json.Unmarshal(data, &config); err != nil {
			return nil, ErrParseJSON.
				WithDetail("path", path).
				WithDetail("reason", err.Error()).
				WithCause(err)
		}

		return config, nil
	}

	return nil, ErrNoConfigSource
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
