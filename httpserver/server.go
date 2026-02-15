package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
)

type Module struct {
	handler  http.Handler
	cfg      Config
	listener net.Listener
	server   *http.Server
	errCh    chan error
}

func NewModule(handler http.Handler, cfg Config) *Module {
	return &Module{
		handler: handler,
		cfg:     cfg.withDefaults(),
		errCh:   make(chan error, 1),
	}
}

func (m *Module) Name() string { return "httpserver" }

func (m *Module) Init(_ context.Context) error {
	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("httpserver: listen %s: %w", addr, err)
	}

	m.listener = ln
	m.server = &http.Server{
		Handler:      m.handler,
		ReadTimeout:  m.cfg.ReadTimeout,
		WriteTimeout: m.cfg.WriteTimeout,
		IdleTimeout:  m.cfg.IdleTimeout,
	}

	return nil
}

func (m *Module) Start(_ context.Context) error {
	go func() {
		err := m.server.Serve(m.listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			m.errCh <- err
		}
	}()

	return nil
}

func (m *Module) Stop(ctx context.Context) error {
	if m.server == nil {
		return nil
	}

	return m.server.Shutdown(ctx)
}

func (m *Module) Err() <-chan error {
	return m.errCh
}

func (m *Module) Addr() string {
	if m.listener != nil {
		return m.listener.Addr().String()
	}

	return ""
}
