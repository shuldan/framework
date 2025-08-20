package cli

import (
	"github.com/shuldan/framework/pkg/contracts"
)

type module struct{}

func (m *module) Name() string {
	return "cli"
}

func (m *module) Register(container contracts.DIContainer) error {
	return container.Factory(
		contracts.CliModuleName,
		func(c contracts.DIContainer) (interface{}, error) {
			r := NewRegistry()
			consoleInstance, err := New(r)
			if err != nil {
				return nil, err
			}

			err = consoleInstance.Register(NewHelpCommand(r))
			if err != nil {
				return nil, err
			}

			return consoleInstance, nil
		},
	)
}

func (m *module) Start(ctx contracts.AppContext) error {
	c, err := ctx.Container().Resolve(contracts.CliModuleName)
	if err != nil {
		return err
	}

	cInst, ok := c.(contracts.Cli)
	if !ok {
		return ErrInvalidConsoleInstance
	}

	return cInst.Run(ctx)
}

func (m *module) Stop(contracts.AppContext) error {
	return nil
}
