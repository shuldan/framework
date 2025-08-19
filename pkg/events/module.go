package events

import (
	"github.com/shuldan/framework/pkg/contracts"
)

type module struct{}

func (m *module) Name() string {
	return contracts.EventBusModuleName
}

func (m *module) Register(container contracts.DIContainer) error {
	return container.Factory(
		contracts.EventBusModuleName,
		func(c contracts.DIContainer) (interface{}, error) {
			b := New()
			b.WithPanicHandler(&defaultPanicHandler{})
			b.WithErrorHandler(&defaultErrorHandler{})
			return b, nil
		},
	)
}

func (m *module) Start(ctx contracts.AppContext) error {
	return nil
}

func (m *module) Stop(ctx contracts.AppContext) error {
	b, err := ctx.Container().Resolve(contracts.EventBusModuleName)
	if err != nil {
		return ErrBusNotFound.WithCause(err)
	}

	busInst, ok := b.(contracts.Bus)
	if !ok {
		return ErrInvalidBusInstance
	}

	return busInst.Close()
}
