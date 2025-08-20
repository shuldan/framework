package logger

import (
	"github.com/shuldan/framework/pkg/contracts"
)

type module struct {
	opts []Option
}

func (m *module) Name() string {
	return contracts.LoggerModuleName
}

func (m *module) Register(container contracts.DIContainer) error {
	return container.Factory(
		contracts.LoggerModuleName,
		func(c contracts.DIContainer) (interface{}, error) {
			return NewLogger(m.opts...)
		},
	)
}

func (m *module) Start(ctx contracts.AppContext) error {
	return nil
}

func (m *module) Stop(ctx contracts.AppContext) error {
	return nil
}
