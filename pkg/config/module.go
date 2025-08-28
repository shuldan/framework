package config

import (
	"fmt"

	"github.com/shuldan/framework/pkg/contracts"
)

type module struct {
	loader Loader
}

func (m *module) Name() string {
	return contracts.ConfigModuleName
}

func (m *module) Register(container contracts.DIContainer) error {
	return container.Factory(contracts.ConfigModuleName, func(c contracts.DIContainer) (interface{}, error) {
		values, err := m.loader.Load()
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
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
