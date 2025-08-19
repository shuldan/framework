package console

import (
	"github.com/shuldan/framework/pkg/application"
)

type Module struct{}

func NewModule() application.Module {
	return &Module{}
}

func (m *Module) Name() string {
	return "console"
}

func (m *Module) Register(container application.Container) error {
	container.Factory(
		"console",
		func(c application.Container) (interface{}, error) {
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

	return nil
}

func (m *Module) Start(ctx application.Context) error {
	c, err := ctx.Container().Resolve("console")
	if err != nil {
		return err
	}

	cInst, ok := c.(Console)
	if !ok {
		return ErrInvalidConsoleInstance
	}

	return cInst.Run(ctx)
}

func (m *Module) Stop(ctx application.Context) error {
	return nil
}
