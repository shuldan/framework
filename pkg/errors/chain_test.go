package errors

import (
	"context"
	"testing"
)

func TestChainErrorHandler_NewChainErrorHandler(t *testing.T) {
	handler := NewChainErrorHandler()
	if handler == nil {
		t.Fatal("handler should not be nil")
	}
	if len(handler.handlers) != 0 {
		t.Error("handlers should be empty initially")
	}
}

func TestChainErrorHandler_Handle_NilError(t *testing.T) {
	handler := NewChainErrorHandler()
	err := handler.Handle(context.Background(), nil)
	if err != nil {
		t.Error("should return nil for nil error")
	}
}
