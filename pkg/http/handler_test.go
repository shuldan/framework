package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestErrorHandler(t *testing.T) {
	t.Parallel()

	config := NewErrorHandlerConfig()
	logger := &mockLogger{}
	handler := NewErrorHandler(config, logger)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	ctx := NewHTTPContext(w, req, logger)

	testCtx := context.WithValue(context.Background(), ContextKey, ctx)

	err := fmt.Errorf("test error")
	if handleErr := handler.Handle(testCtx, err); handleErr != nil {
		t.Fatalf("Error handler failed: %v", handleErr)
	}

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("JSON decode failed: %v", err)
	}

	errorData, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Error object not found in response")
	}

	if errorData["message"] != "Internal server error" {
		t.Errorf("Expected message 'Internal server error', got %v", errorData["message"])
	}

	if errorData["request_id"] == "" {
		t.Error("Request ID not set in error response")
	}
}

func TestErrorHandlerNilError(t *testing.T) {
	t.Parallel()

	config := NewErrorHandlerConfig()
	logger := &mockLogger{}
	handler := NewErrorHandler(config, logger)

	if err := handler.Handle(context.Background(), nil); err != nil {
		t.Errorf("Expected nil for nil error, got %v", err)
	}
}

func TestErrorHandlerNoHTTPContext(t *testing.T) {
	t.Parallel()

	config := NewErrorHandlerConfig()
	logger := &mockLogger{}
	handler := NewErrorHandler(config, logger)

	err := fmt.Errorf("test error")
	if handleErr := handler.Handle(context.Background(), err); !errors.Is(handleErr, err) {
		t.Errorf("Expected original error when no HTTP context, got %v", handleErr)
	}

	messages := logger.getMessages()
	found := false
	for _, msg := range messages {
		if strings.Contains(msg, "HTTP context not found") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected error log about missing HTTP context")
	}
}

func TestErrorHandlerConfig(t *testing.T) {
	t.Parallel()

	config := NewErrorHandlerConfig().
		WithStatusCode("TEST", http.StatusTeapot).
		WithUserMessage("TEST", "Test message").
		WithLogLevel("debug").
		WithShowStackTrace(true).
		WithShowDetails(true)

	if config.StatusCodeMap()["TEST"] != http.StatusTeapot {
		t.Error("Status code not set correctly")
	}

	if config.UserMessageMap()["TEST"] != "Test message" {
		t.Error("User message not set correctly")
	}

	if config.LogLevel() != "debug" {
		t.Error("Log level not set correctly")
	}

	if !config.ShowStackTrace() {
		t.Error("Show stack trace not set correctly")
	}

	if !config.ShowDetails() {
		t.Error("Show details not set correctly")
	}
}
