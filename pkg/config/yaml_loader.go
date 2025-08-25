package config

import (
	"github.com/goccy/go-yaml"
	"os"
)

type YamlConfigLoader struct {
	paths []string
}

func (l *YamlConfigLoader) Load() (map[string]any, error) {
	for _, path := range l.paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var config map[string]any
		if err = yaml.UnmarshalWithOptions(data, &config, yaml.UseJSONUnmarshaler()); err != nil {
			return nil, ErrParseYAML.
				WithDetail("path", path).
				WithDetail("reason", err.Error()).
				WithCause(err)
		}

		return config, nil
	}

	return nil, ErrNoConfigSource
}
