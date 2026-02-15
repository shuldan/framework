package queueworker

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

type noopLogger struct{}

func (noopLogger) Info(string, ...any)  {}
func (noopLogger) Error(string, ...any) {}

type Registration struct {
	Name string
	Run  func(ctx context.Context) error
}

type Module struct {
	logger        Logger
	registrations []Registration
	errCh         chan error
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

func NewModule(log Logger) *Module {
	return &Module{
		logger: ensureLog(log),
		errCh:  make(chan error, 1),
	}
}

func (m *Module) Register(reg Registration) {
	m.registrations = append(m.registrations, reg)

	m.logger.Info("queue consumer registered",
		"name", reg.Name,
	)
}

func (m *Module) Name() string { return "queueworker" }

func (m *Module) Init(_ context.Context) error { return nil }

func (m *Module) Start(_ context.Context) error {
	if len(m.registrations) == 0 {
		m.logger.Info("no queue consumers registered")
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	for _, reg := range m.registrations {
		m.wg.Add(1)

		go m.runConsumer(ctx, reg)
	}

	m.logger.Info("queue workers started",
		"count", len(m.registrations),
	)

	return nil
}

func (m *Module) Stop(_ context.Context) error {
	if m.cancel != nil {
		m.cancel()
	}

	m.wg.Wait()
	m.logger.Info("queue workers stopped")

	return nil
}

func (m *Module) Err() <-chan error {
	return m.errCh
}

func (m *Module) ConsumerCount() int {
	return len(m.registrations)
}

func (m *Module) runConsumer(
	ctx context.Context, reg Registration,
) {
	defer m.wg.Done()

	m.logger.Info("consumer starting", "name", reg.Name)

	err := reg.Run(ctx)
	if err == nil || isContextErr(err) {
		return
	}

	m.logger.Error("consumer failed",
		"name", reg.Name, "error", err,
	)

	select {
	case m.errCh <- fmt.Errorf("consumer %q: %w", reg.Name, err):
	default:
	}
}

func isContextErr(err error) bool {
	return errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded)
}

func ensureLog(log Logger) Logger {
	if log == nil {
		return noopLogger{}
	}

	return log
}
