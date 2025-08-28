package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
)

type YamlConfigLoader struct {
	paths []string
}

func (l *YamlConfigLoader) Load() (map[string]any, error) {
	for _, path := range l.paths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		absPath = filepath.Clean(absPath)

		wd, err := os.Getwd()
		if err != nil {
			wd = "."
		}
		secureBase, err := filepath.Abs(wd)
		if err != nil {
			secureBase = "/"
		}
		secureBase = filepath.Clean(secureBase)

		if !strings.HasPrefix(absPath, secureBase+string(filepath.Separator)) {
			continue
		}

		if strings.Contains(absPath, "..") {
			continue
		}

		data, err := os.ReadFile(absPath)
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
