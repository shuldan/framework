package eventbus

import (
	"context"

	"github.com/shuldan/events"
)

type Module struct {
	dispatcher *events.Dispatcher
}

func NewModule(dispatcher *events.Dispatcher) *Module {
	return &Module{
		dispatcher: dispatcher,
	}
}

func (m *Module) Dispatcher() *events.Dispatcher {
	return m.dispatcher
}

func (m *Module) Name() string { return "eventbus" }

func (m *Module) Init(_ context.Context) error { return nil }

func (m *Module) Start(_ context.Context) error { return nil }

func (m *Module) Stop(ctx context.Context) error {
	return m.dispatcher.Close(ctx)
}
