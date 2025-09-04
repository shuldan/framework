package config

import "github.com/shuldan/framework/pkg/contracts"

var _ Loader = (*envConfigLoader)(nil)
var _ Loader = (*YamlConfigLoader)(nil)
var _ Loader = (*jsonConfigLoader)(nil)

func NewEnvConfigLoader(prefix string) Loader {
	return &envConfigLoader{prefix: prefix}
}

func NewYamlConfigLoader(paths ...string) Loader {
	return &YamlConfigLoader{paths: paths}
}

func NewJSONConfigLoader(paths ...string) Loader {
	return &jsonConfigLoader{paths: paths}
}

func NewChainLoader(loaders ...Loader) Loader {
	return &chainLoader{loaders: loaders}
}

func NewMapConfig(values map[string]any) contracts.Config {
	return &MapConfig{values: values}
}

func NewModule(loader Loader) contracts.AppModule {
	return &module{loader: newTemplatedLoader(loader)}
}
