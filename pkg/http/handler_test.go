package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shuldan/framework/pkg/errors"
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

func TestErrorHandlerWithDetails(t *testing.T) {
	t.Parallel()

	config := NewErrorHandlerConfig().WithShowDetails(true)
	logger := &mockLogger{}
	handler := NewErrorHandler(config, logger)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	ctx := NewHTTPContext(w, req, logger)

	testCtx := context.WithValue(context.Background(), ContextKey, ctx)

	detailErr := errors.ErrValidation.WithDetail("field", "email").WithDetail("reason", "invalid format")
	if handleErr := handler.Handle(testCtx, detailErr); handleErr != nil {
		t.Fatalf("Error handler failed: %v", handleErr)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("JSON decode failed: %v", err)
	}

	errorData := response["error"].(map[string]interface{})
	details := errorData["details"].(map[string]interface{})

	if details["field"] != "email" {
		t.Errorf("Expected field detail 'email', got %v", details["field"])
	}
}

func TestErrorHandlerWithStackTrace(t *testing.T) {
	t.Parallel()

	config := NewErrorHandlerConfig().WithShowStackTrace(true)
	logger := &mockLogger{}
	handler := NewErrorHandler(config, logger)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	ctx := NewHTTPContext(w, req, logger)

	testCtx := context.WithValue(context.Background(), ContextKey, ctx)

	internalErr := errors.ErrInternal.WithCause(fmt.Errorf("database connection failed"))
	if handleErr := handler.Handle(testCtx, internalErr); handleErr != nil {
		t.Fatalf("Error handler failed: %v", handleErr)
	}

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("JSON decode failed: %v", err)
	}

	errorData := response["error"].(map[string]interface{})
	if _, hasStack := errorData["stack_trace"]; !hasStack {
		t.Error("Expected stack trace in error response for internal error")
	}
}

func TestErrorHandlerDifferentErrorTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		statusCode int
		errorCode  string
	}{
		{"Validation Error", errors.ErrValidation, http.StatusBadRequest, string(errors.ErrValidation.Code)},
		{"Auth Error", errors.ErrAuth, http.StatusUnauthorized, string(errors.ErrAuth.Code)},
		{"Permission Error", errors.ErrPermission, http.StatusForbidden, string(errors.ErrPermission.Code)},
		{"Not Found Error", errors.ErrNotFound, http.StatusNotFound, string(errors.ErrNotFound.Code)},
		{"Conflict Error", errors.ErrConflict, http.StatusConflict, string(errors.ErrConflict.Code)},
		{"Business Error", errors.ErrBusiness, http.StatusUnprocessableEntity, string(errors.ErrBusiness.Code)},
		{"Timeout Error", errors.ErrTimeout, http.StatusRequestTimeout, string(errors.ErrTimeout.Code)},
		{"Unavailable Error", errors.ErrUnavailable, http.StatusServiceUnavailable, string(errors.ErrUnavailable.Code)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewErrorHandlerConfig()
			logger := &mockLogger{}
			handler := NewErrorHandler(config, logger)

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			ctx := NewHTTPContext(w, req, logger)

			testCtx := context.WithValue(context.Background(), ContextKey, ctx)

			if handleErr := handler.Handle(testCtx, tt.err); handleErr != nil {
				t.Fatalf("Error handler failed: %v", handleErr)
			}

			if w.Code != tt.statusCode {
				t.Errorf("Expected status %d, got %d", tt.statusCode, w.Code)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("JSON decode failed: %v", err)
			}

			errorData := response["error"].(map[string]interface{})
			if errorData["code"] != tt.errorCode {
				t.Errorf("Expected error code %s, got %v", tt.errorCode, errorData["code"])
			}
		})
	}
}

func TestErrorHandlerAlreadySentResponse(t *testing.T) {
	t.Parallel()

	config := NewErrorHandlerConfig()
	logger := &mockLogger{}
	handler := NewErrorHandler(config, logger)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	ctx := NewHTTPContext(w, req, logger)

	if err := ctx.Status(200).JSON(map[string]string{"already": "sent"}); err != nil {
		t.Fatalf("Initial response failed: %v", err)
	}

	testCtx := context.WithValue(context.Background(), ContextKey, ctx)
	err := fmt.Errorf("test error")

	if handleErr := handler.Handle(testCtx, err); handleErr != nil {
		t.Errorf("Error handler should handle already sent response gracefully: %v", handleErr)
	}

	messages := logger.getMessages()
	found := false
	for _, msg := range messages {
		if strings.Contains(msg, "Response already sent") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected warning about response already sent")
	}
}

func TestErrorHandlerCustomStatusCodes(t *testing.T) {
	t.Parallel()

	config := NewErrorHandlerConfig().
		WithStatusCode("CUSTOM", http.StatusTeapot).
		WithUserMessage("CUSTOM", "I'm a teapot")

	logger := &mockLogger{}
	handler := NewErrorHandler(config, logger)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	ctx := NewHTTPContext(w, req, logger)

	testCtx := context.WithValue(context.Background(), ContextKey, ctx)
	customErr := fmt.Errorf("CUSTOM: %v", "custom error occurred")

	if handleErr := handler.Handle(testCtx, customErr); handleErr != nil {
		t.Fatalf("Error handler failed: %v", handleErr)
	}

	if w.Code != http.StatusTeapot {
		t.Errorf("Expected status 418, got %d", w.Code)
	}
}

func TestErrorHandlerJSONMarshalError(t *testing.T) {
	t.Parallel()

	config := NewErrorHandlerConfig()
	logger := &mockLogger{}
	handler := NewErrorHandler(config, logger)

	req := httptest.NewRequest("GET", "/test", nil)
	w := &brokenResponseWriter{httptest.NewRecorder()}
	ctx := NewHTTPContext(w, req, logger)

	testCtx := context.WithValue(context.Background(), ContextKey, ctx)
	err := fmt.Errorf("test error")

	if handleErr := handler.Handle(testCtx, err); handleErr == nil {
		t.Error("Expected error when JSON marshaling fails")
	}
}

type brokenResponseWriter struct {
	*httptest.ResponseRecorder
}

func (b *brokenResponseWriter) Write([]byte) (int, error) {
	return 0, fmt.Errorf("write error")
}
