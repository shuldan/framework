package config

import (
	"reflect"

	"github.com/shuldan/framework/pkg/contracts"
)

const ModuleName = "config"

type module struct {
	loader Loader
}

func NewModule(envPrefix string, configPaths ...string) contracts.AppModule {
	loaders := []Loader{
		NewYamlConfigLoader(configPaths...),
		NewJSONConfigLoader(configPaths...),
		NewEnvConfigLoader(envPrefix),
	}

	return &module{loader: newTemplatedLoader(NewChainLoader(loaders...))}
}

func (m *module) Name() string {
	return ModuleName
}

func (m *module) Register(container contracts.DIContainer) error {
	return container.Factory(reflect.TypeOf((*contracts.Config)(nil)).Elem(), func(c contracts.DIContainer) (interface{}, error) {
		values, err := m.loader.Load()
		if err != nil {
			return nil, err
		}
		return NewMapConfig(values), nil
	})
}

func (m *module) Start(_ contracts.AppContext) error {
	return nil
}

func (m *module) Stop(_ contracts.AppContext) error {
	return nil
}
