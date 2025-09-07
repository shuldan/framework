package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

var _ Loader = (*jsonConfigLoader)(nil)

func NewJSONConfigLoader(paths ...string) Loader {
	return &jsonConfigLoader{paths: paths}
}

type jsonConfigLoader struct {
	paths []string
}

func (l *jsonConfigLoader) Load() (map[string]any, error) {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	secureBase, err := filepath.Abs(wd)
	if err != nil {
		secureBase = "/"
	}
	secureBase = filepath.Clean(secureBase)

	for _, path := range l.paths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		absPath = filepath.Clean(absPath)

		if !strings.HasPrefix(absPath, secureBase+string(filepath.Separator)) {
			continue
		}

		if strings.Contains(absPath, "..") {
			continue
		}

		if !fileExists(absPath) {
			continue
		}

		data, err := os.ReadFile(absPath)
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

	return nil, ErrNoConfigSource.WithDetail("loader", "json")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
