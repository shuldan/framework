package errors

import (
	"context"
	"errors"
	"testing"
)

type contextKey string

const testContextKey contextKey = "test-key"

type mockHandler struct {
	shouldHandle bool
	callCount    int
}

func (m *mockHandler) Handle(ctx context.Context, err error) error {
	m.callCount++
	if m.shouldHandle {
		return nil
	}
	return err
}

func TestChainErrorHandler_NewChainErrorHandler(t *testing.T) {
	handler := NewChainErrorHandler()
	if handler == nil {
		t.Fatal("handler should not be nil")
	}
	if len(handler.handlers) != 0 {
		t.Error("handlers should be empty initially")
	}
}

func TestChainErrorHandler_NewChainErrorHandler_WithHandlers(t *testing.T) {
	h1 := &mockHandler{}
	h2 := &mockHandler{}

	handler := NewChainErrorHandler(h1, h2)
	if handler == nil {
		t.Fatal("handler should not be nil")
	}
	if len(handler.handlers) != 2 {
		t.Errorf("expected 2 handlers, got %d", len(handler.handlers))
	}
}

func TestChainErrorHandler_Add(t *testing.T) {
	handler := NewChainErrorHandler()
	h1 := &mockHandler{}
	h2 := &mockHandler{}

	result := handler.Add(h1).Add(h2)

	if result != handler {
		t.Error("Add should return the same handler for chaining")
	}

	if len(handler.handlers) != 2 {
		t.Errorf("expected 2 handlers, got %d", len(handler.handlers))
	}
}

func TestChainErrorHandler_Handle_NilError(t *testing.T) {
	handler := NewChainErrorHandler()
	err := handler.Handle(context.Background(), nil)
	if err != nil {
		t.Error("should return nil for nil error")
	}
}

func TestChainErrorHandler_Handle_SingleHandlerSuccess(t *testing.T) {
	h1 := &mockHandler{shouldHandle: true}
	handler := NewChainErrorHandler(h1)

	testErr := errors.New("test error")
	result := handler.Handle(context.Background(), testErr)

	if result != nil {
		t.Error("should return nil when handler successfully handles error")
	}
	if h1.callCount != 1 {
		t.Errorf("expected handler to be called once, got %d", h1.callCount)
	}
}

func TestChainErrorHandler_Handle_SingleHandlerFail(t *testing.T) {
	h1 := &mockHandler{shouldHandle: false}
	handler := NewChainErrorHandler(h1)

	testErr := errors.New("test error")
	result := handler.Handle(context.Background(), testErr)

	if !errors.Is(result, testErr) {
		t.Error("should return original error when handler fails")
	}
	if h1.callCount != 1 {
		t.Errorf("expected handler to be called once, got %d", h1.callCount)
	}
}

func TestChainErrorHandler_Handle_MultipleHandlers_FirstSucceeds(t *testing.T) {
	h1 := &mockHandler{shouldHandle: true}
	h2 := &mockHandler{shouldHandle: false}
	h3 := &mockHandler{shouldHandle: true}

	handler := NewChainErrorHandler(h1, h2, h3)

	testErr := errors.New("test error")
	result := handler.Handle(context.Background(), testErr)

	if result != nil {
		t.Error("should return nil when first handler succeeds")
	}
	if h1.callCount != 1 {
		t.Errorf("expected first handler to be called once, got %d", h1.callCount)
	}
	if h2.callCount != 0 {
		t.Errorf("expected second handler not to be called, got %d", h2.callCount)
	}
	if h3.callCount != 0 {
		t.Errorf("expected third handler not to be called, got %d", h3.callCount)
	}
}

func TestChainErrorHandler_Handle_MultipleHandlers_SecondSucceeds(t *testing.T) {
	h1 := &mockHandler{shouldHandle: false}
	h2 := &mockHandler{shouldHandle: true}
	h3 := &mockHandler{shouldHandle: false}

	handler := NewChainErrorHandler(h1, h2, h3)

	testErr := errors.New("test error")
	result := handler.Handle(context.Background(), testErr)

	if result != nil {
		t.Error("should return nil when second handler succeeds")
	}
	if h1.callCount != 1 {
		t.Errorf("expected first handler to be called once, got %d", h1.callCount)
	}
	if h2.callCount != 1 {
		t.Errorf("expected second handler to be called once, got %d", h2.callCount)
	}
	if h3.callCount != 0 {
		t.Errorf("expected third handler not to be called, got %d", h3.callCount)
	}
}

func TestChainErrorHandler_Handle_AllHandlersFail(t *testing.T) {
	h1 := &mockHandler{shouldHandle: false}
	h2 := &mockHandler{shouldHandle: false}
	h3 := &mockHandler{shouldHandle: false}

	handler := NewChainErrorHandler(h1, h2, h3)

	testErr := errors.New("test error")
	result := handler.Handle(context.Background(), testErr)

	if !errors.Is(result, testErr) {
		t.Error("should return original error when all handlers fail")
	}
	if h1.callCount != 1 {
		t.Errorf("expected first handler to be called once, got %d", h1.callCount)
	}
	if h2.callCount != 1 {
		t.Errorf("expected second handler to be called once, got %d", h2.callCount)
	}
	if h3.callCount != 1 {
		t.Errorf("expected third handler to be called once, got %d", h3.callCount)
	}
}

func TestChainErrorHandler_Handle_EmptyChain(t *testing.T) {
	handler := NewChainErrorHandler()

	testErr := errors.New("test error")
	result := handler.Handle(context.Background(), testErr)

	if !errors.Is(result, testErr) {
		t.Error("should return original error when no handlers")
	}
}

func TestChainErrorHandler_Handle_WithContext(t *testing.T) {
	contextHandler := &contextAwareHandler{shouldHandle: false}
	handler := NewChainErrorHandler(contextHandler)

	ctx := context.WithValue(context.Background(), testContextKey, "test-value")
	testErr := errors.New("test error")

	result := handler.Handle(ctx, testErr)

	if !errors.Is(result, testErr) {
		t.Error("should return original error when handler doesn't handle it")
	}

	if !contextHandler.contextReceived {
		t.Error("handler should receive context")
	}
	if contextHandler.receivedValue != "test-value" {
		t.Errorf("expected context value 'test-value', got %v", contextHandler.receivedValue)
	}
}

type contextAwareHandler struct {
	contextReceived bool
	receivedValue   interface{}
	shouldHandle    bool
}

func (c *contextAwareHandler) Handle(ctx context.Context, err error) error {
	c.contextReceived = true
	c.receivedValue = ctx.Value(testContextKey)

	if c.shouldHandle {
		return nil
	}
	return err
}
