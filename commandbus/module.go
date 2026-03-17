package commandbus

import (
	"context"
	"fmt"

	"github.com/shuldan/commands"
)

// Module управляет жизненным циклом командной шины.
type Module struct {
	client *commands.CommandClient
	server *commands.CommandServer
}

// Option — функциональная опция для настройки Module.
type Option func(*Module)

// WithClient добавляет клиент в модуль.
func WithClient(client *commands.CommandClient) Option {
	return func(m *Module) {
		m.client = client
	}
}

// WithServer добавляет сервер в модуль.
func WithServer(server *commands.CommandServer) Option {
	return func(m *Module) {
		m.server = server
	}
}

// NewModule создаёт Module с опциями.
func NewModule(opts ...Option) *Module {
	m := &Module{}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// Name возвращает имя модуля.
func (m *Module) Name() string { return "commandbus" }

func (m *Module) Init(_ context.Context) error {
	return nil
}

// Start открывает клиент и сервер.
func (m *Module) Start(ctx context.Context) error {
	if m.server != nil {
		if err := m.server.Open(ctx); err != nil {
			return fmt.Errorf("commandbus: open server: %w", err)
		}
	}

	if m.client != nil {
		if err := m.client.Open(ctx); err != nil {
			return fmt.Errorf("commandbus: open client: %w", err)
		}
	}

	return nil
}

// Stop закрывает клиент и сервер.
func (m *Module) Stop(ctx context.Context) error {
	var firstErr error

	if m.client != nil {
		if err := m.client.Close(ctx); err != nil {
			firstErr = fmt.Errorf("commandbus: close client: %w", err)
		}
	}

	if m.server != nil {
		if err := m.server.Close(ctx); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("commandbus: close server: %w", err)
		}
	}

	return firstErr
}

// Client возвращает клиент команд.
func (m *Module) Client() *commands.CommandClient {
	return m.client
}

// Server возвращает сервер команд.
func (m *Module) Server() *commands.CommandServer {
	return m.server
}
