package config

import "github.com/shuldan/framework/pkg/errors"

var newConfigCode = errors.WithPrefix("CONFIG")

var (
	ErrNoConfigSource = newConfigCode().New("no valid configuration source found. Loader: {{.loader}}")
	ErrParseYAML      = newConfigCode().New("failed to parse YAML file {{.path}}: {{.reason}}")
	ErrParseJSON      = newConfigCode().New("failed to parse JSON file {{.path}}: {{.reason}}")
	ErrMergeFailed    = newConfigCode().New("failed to merge configuration layers")
)
