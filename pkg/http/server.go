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

type serverConfig struct {
	address           string
	readHeaderTimeout time.Duration
	readTimeout       time.Duration
	writeTimeout      time.Duration
	idleTimeout       time.Duration
	shutdownTimeout   time.Duration
}

func defaultServerConfig() *serverConfig {
	return &serverConfig{
		address:           ":8080",
		readHeaderTimeout: 30 * time.Second,
		readTimeout:       30 * time.Second,
		writeTimeout:      60 * time.Second,
		idleTimeout:       90 * time.Second,
		shutdownTimeout:   30 * time.Second,
	}
}

type httpServer struct {
	server  *http.Server
	router  contracts.HTTPRouter
	logger  contracts.Logger
	running bool
	mu      sync.RWMutex
	config  *serverConfig
}

func NewServer(router contracts.HTTPRouter, logger contracts.Logger, options ...ServerOption) (contracts.HTTPServer, error) {
	if router == nil {
		return nil, ErrInvalidHTTPRouterInstance
	}
	if logger == nil {
		return nil, ErrHTTPRouterNotFound
	}

	config := defaultServerConfig()
	for _, opt := range options {
		opt(config)
	}

	return &httpServer{
		router: router,
		logger: logger,
		config: config,
	}, nil
}

func (s *httpServer) Start(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return ErrServerAlreadyRunning
	}

	listener, err := net.Listen("tcp", s.config.address)
	if err != nil {
		return ErrServerStart.WithCause(err).WithDetail("addr", s.config.address)
	}

	s.server = &http.Server{
		Addr:              listener.Addr().String(),
		Handler:           s.router,
		ReadHeaderTimeout: s.config.readHeaderTimeout,
		ReadTimeout:       s.config.readTimeout,
		WriteTimeout:      s.config.writeTimeout,
		IdleTimeout:       s.config.idleTimeout,
	}

	s.running = true

	go func() {
		if err := s.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			if s.logger != nil {
				s.logger.Error("server error", "error", err)
			}
		}
	}()

	if s.logger != nil {
		s.logger.Info("HTTP server started", "addr", s.config.address)
	}

	return nil
}

func (s *httpServer) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	timeoutCtx, cancel := context.WithTimeout(ctx, s.config.shutdownTimeout)
	defer cancel()

	if !s.running || s.server == nil {
		return nil
	}

	if err := s.server.Shutdown(timeoutCtx); err != nil {
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
	if !s.running {
		return s.config.address
	}
	return s.server.Addr
}

func (s *httpServer) Handler() http.Handler {
	return s.router
}
