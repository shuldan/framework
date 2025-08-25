package config

import "github.com/shuldan/framework/pkg/contracts"

var _ Loader = (*EnvConfigLoader)(nil)
var _ Loader = (*YamlConfigLoader)(nil)
var _ Loader = (*JSONConfigLoader)(nil)

func NewEnvConfigLoader(prefix string) Loader {
	return &EnvConfigLoader{prefix: prefix}
}

func NewYamlConfigLoader(paths ...string) *YamlConfigLoader {
	return &YamlConfigLoader{paths: paths}
}

func NewJSONConfigLoader(paths ...string) *JSONConfigLoader {
	return &JSONConfigLoader{paths: paths}
}

func NewChainLoader(loaders ...Loader) Loader {
	return &ChainLoader{loaders: loaders}
}

func NewMapConfig(values map[string]any) contracts.Config {
	return &MapConfig{values: values}
}

func NewModule(loader Loader) contracts.AppModule {
	return &module{loader: loader}
}
