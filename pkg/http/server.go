package http

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/shuldan/framework/pkg/contracts"
)

type httpServer struct {
	server  *http.Server
	router  contracts.HTTPRouter
	addr    string
	logger  contracts.Logger
	running bool
	mu      sync.RWMutex
}

func NewServer(addr string, router contracts.HTTPRouter, logger contracts.Logger) (contracts.HTTPServer, error) {
	if router == nil {
		return nil, ErrInvalidHTTPRouterInstance
	}
	if logger == nil {
		return nil, ErrHTTPRouterNotFound
	}
	if addr == "" {
		addr = ":8080"
	}

	return &httpServer{
		addr:   addr,
		router: router,
		logger: logger,
	}, nil
}

func (s *httpServer) Start(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return ErrServerAlreadyRunning
	}

	s.server = &http.Server{
		Addr:              s.addr,
		Handler:           s.router,
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       90 * time.Second,
	}

	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return ErrServerStart.WithCause(err).WithDetail("addr", s.addr)
	}

	s.addr = listener.Addr().String()
	s.running = true

	go func() {
		if err := s.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			if s.logger != nil {
				s.logger.Error("server error", "error", err)
			}
		}
	}()

	if s.logger != nil {
		s.logger.Info("HTTP server started", "addr", s.addr)
	}

	return nil
}

func (s *httpServer) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running || s.server == nil {
		return nil
	}

	if err := s.server.Shutdown(ctx); err != nil {
		return ErrServerStop.WithCause(err)
	}

	s.running = false
	if s.logger != nil {
		s.logger.Info("HTTP server stopped")
	}

	return nil
}

func (s *httpServer) Addr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.addr
}

func (s *httpServer) Handler() http.Handler {
	return s.router
}
