package events

import (
	"testing"
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
