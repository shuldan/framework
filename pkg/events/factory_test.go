package events

import (
	"testing"

	"github.com/shuldan/framework/pkg/contracts"
)

func TestNew_WithOptions(t *testing.T) {
	t.Parallel()

	panicHandler := &mockPanicHandler{}
	errorHandler := &mockErrorHandler{}

	bus := New(
		WithPanicHandler(panicHandler),
		WithErrorHandler(errorHandler),
		WithAsyncMode(true),
		WithWorkerCount(3),
	)

	if bus == nil {
		t.Fatal("expected non-nil eventBus")
	}

	err := bus.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestNew_DefaultHandlers(t *testing.T) {
	t.Parallel()

	bus := New()

	if bus == nil {
		t.Fatal("expected non-nil eventBus")
	}

	err := bus.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestNewModule(t *testing.T) {
	t.Parallel()

	module := NewModule()

	if module == nil {
		t.Fatal("expected non-nil module")
	}

	if module.Name() != contracts.EventBusModuleName {
		t.Errorf("expected module name %s, got %s", contracts.EventBusModuleName, module.Name())
	}
}
