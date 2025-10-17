package cli

import (
	"os"
	"reflect"

	"github.com/shuldan/framework/pkg/contracts"
)

const ModuleName = "cli"

type module struct{}

func NewModule() contracts.AppModule {
	return &module{}
}

func (m *module) Name() string {
	return ModuleName
}

func (m *module) Register(container contracts.DIContainer) error {
	return container.Factory(
		reflect.TypeOf((*contracts.Cli)(nil)).Elem(),
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
	c, err := ctx.Container().Resolve(reflect.TypeOf((*contracts.Cli)(nil)).Elem())
	if err != nil {
		return err
	}

	cliInst, ok := c.(contracts.Cli)
	if !ok {
		return ErrInvalidConsoleInstance
	}

	registry := ctx.AppRegistry()
	for _, module := range registry.All() {
		if provider, ok := module.(contracts.CliCommandProvider); ok {
			commands, err := provider.CliCommands(ctx)
			if err != nil {
				return ErrFailedRegisterCommand.WithCause(err).WithDetail("module", module.Name())
			}
			for _, cmd := range commands {
				if err := cliInst.Register(cmd); err != nil {
					return ErrFailedRegisterCommand.WithCause(err).WithDetail("module", module.Name())
				}
			}
		}
	}

	return cliInst.Run(NewContext(ctx, os.Stdin, os.Stdout, os.Args[1:]))
}

func (m *module) Stop(contracts.AppContext) error {
	return nil
}
