package commandbus

import (
	"context"

	"github.com/shuldan/commands"
)

// Module управляет жизненным циклом командной шины.
type Module struct {
	dispatcher *commands.Dispatcher
}

// NewModule создаёт Module с локальным диспетчером.
func NewModule(cfg Config) *Module {
	return &Module{
		dispatcher: commands.New(buildOpts(cfg)...),
	}
}

// Dispatcher возвращает локальный диспетчер команд.
func (m *Module) Dispatcher() *commands.Dispatcher {
	return m.dispatcher
}

// Name возвращает имя модуля.
func (m *Module) Name() string { return "commandbus" }

// Init инициализирует модуль.
func (m *Module) Init(_ context.Context) error { return nil }

// Start запускает модуль.
func (m *Module) Start(_ context.Context) error { return nil }

// Stop останавливает модуль.
func (m *Module) Stop(ctx context.Context) error {
	return m.dispatcher.Close(ctx)
}
