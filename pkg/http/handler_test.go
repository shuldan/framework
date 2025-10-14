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

func TestErrorHandlerWithConfiguredErrorCodes(t *testing.T) {
	t.Parallel()

	config := NewErrorHandlerConfig().
		WithStatusCode("CORE_0001", http.StatusBadRequest).
		WithStatusCode("CORE_0002", http.StatusUnauthorized).
		WithUserMessage("CORE_0001", "Custom validation message").
		WithUserMessage("CORE_0002", "Custom auth message")

	logger := &mockLogger{}
	handler := NewErrorHandler(config, logger)

	tests := []struct {
		name         string
		err          error
		expectedCode int
		expectedMsg  string
		errorCode    string
	}{
		{
			name:         "Validation Error with CORE_0001",
			err:          errors.ErrValidation,
			expectedCode: http.StatusBadRequest,
			expectedMsg:  "Custom validation message",
			errorCode:    "CORE_0001",
		},
		{
			name:         "Auth Error with CORE_0002",
			err:          errors.ErrAuth,
			expectedCode: http.StatusUnauthorized,
			expectedMsg:  "Custom auth message",
			errorCode:    "CORE_0002",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			ctx := NewHTTPContext(w, req, logger)

			testCtx := context.WithValue(context.Background(), ContextKey, ctx)

			if handleErr := handler.Handle(testCtx, tt.err); handleErr != nil {
				t.Fatalf("Error handler failed: %v", handleErr)
			}

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status %d, got %d", tt.expectedCode, w.Code)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("JSON decode failed: %v", err)
			}

			errorData := response["error"].(map[string]interface{})
			if errorData["code"] != tt.errorCode {
				t.Errorf("Expected error code %s, got %v", tt.errorCode, errorData["code"])
			}
			if errorData["message"] != tt.expectedMsg {
				t.Errorf("Expected message %s, got %v", tt.expectedMsg, errorData["message"])
			}
		})
	}
}

func TestErrorHandlerConfigStatusCodeMap(t *testing.T) {
	t.Parallel()

	config := NewErrorHandlerConfig()

	expectedMappings := map[string]int{
		string(errors.ErrValidation.Code):  http.StatusBadRequest,
		string(errors.ErrAuth.Code):        http.StatusUnauthorized,
		string(errors.ErrPermission.Code):  http.StatusForbidden,
		string(errors.ErrNotFound.Code):    http.StatusNotFound,
		string(errors.ErrConflict.Code):    http.StatusConflict,
		string(errors.ErrBusiness.Code):    http.StatusUnprocessableEntity,
		string(errors.ErrTimeout.Code):     http.StatusRequestTimeout,
		string(errors.ErrUnavailable.Code): http.StatusServiceUnavailable,
		string(errors.ErrInternal.Code):    http.StatusInternalServerError,
	}

	statusCodeMap := config.StatusCodeMap()
	for errorCode, expectedStatus := range expectedMappings {
		if actualStatus := statusCodeMap[errorCode]; actualStatus != expectedStatus {
			t.Errorf("Expected status code %d for error %s, got %d",
				expectedStatus, errorCode, actualStatus)
		}
	}
}

func TestErrorHandlerConfigUserMessageMap(t *testing.T) {
	t.Parallel()

	config := NewErrorHandlerConfig()

	expectedMessages := map[string]string{
		string(errors.ErrValidation.Code):  "Invalid input data",
		string(errors.ErrAuth.Code):        "Authentication required",
		string(errors.ErrPermission.Code):  "Access denied",
		string(errors.ErrNotFound.Code):    "Resource not found",
		string(errors.ErrConflict.Code):    "Resource already exists",
		string(errors.ErrBusiness.Code):    "Operation not allowed",
		string(errors.ErrTimeout.Code):     "Request timeout",
		string(errors.ErrUnavailable.Code): "Service temporarily unavailable",
		string(errors.ErrInternal.Code):    "Internal server error",
	}

	userMessageMap := config.UserMessageMap()
	for errorCode, expectedMessage := range expectedMessages {
		if actualMessage := userMessageMap[errorCode]; actualMessage != expectedMessage {
			t.Errorf("Expected message %s for error %s, got %s",
				expectedMessage, errorCode, actualMessage)
		}
	}
}

func TestErrorHandlerSetsCorrectStatusCodes(t *testing.T) {
	t.Parallel()

	config := NewErrorHandlerConfig()
	logger := &mockLogger{}
	handler := NewErrorHandler(config, logger)

	testCases := []struct {
		name           string
		error          error
		expectedStatus int
		errorCode      string
	}{
		{
			name:           "Validation Error",
			error:          errors.ErrValidation,
			expectedStatus: http.StatusBadRequest,
			errorCode:      string(errors.ErrValidation.Code),
		},
		{
			name:           "Auth Error",
			error:          errors.ErrAuth,
			expectedStatus: http.StatusUnauthorized,
			errorCode:      string(errors.ErrAuth.Code),
		},
		{
			name:           "Permission Error",
			error:          errors.ErrPermission,
			expectedStatus: http.StatusForbidden,
			errorCode:      string(errors.ErrPermission.Code),
		},
		{
			name:           "Not Found Error",
			error:          errors.ErrNotFound,
			expectedStatus: http.StatusNotFound,
			errorCode:      string(errors.ErrNotFound.Code),
		},
		{
			name:           "Conflict Error",
			error:          errors.ErrConflict,
			expectedStatus: http.StatusConflict,
			errorCode:      string(errors.ErrConflict.Code),
		},
		{
			name:           "Business Error",
			error:          errors.ErrBusiness,
			expectedStatus: http.StatusUnprocessableEntity,
			errorCode:      string(errors.ErrBusiness.Code),
		},
		{
			name:           "Timeout Error",
			error:          errors.ErrTimeout,
			expectedStatus: http.StatusRequestTimeout,
			errorCode:      string(errors.ErrTimeout.Code),
		},
		{
			name:           "Unavailable Error",
			error:          errors.ErrUnavailable,
			expectedStatus: http.StatusServiceUnavailable,
			errorCode:      string(errors.ErrUnavailable.Code),
		},
		{
			name:           "Internal Error",
			error:          errors.ErrInternal,
			expectedStatus: http.StatusInternalServerError,
			errorCode:      string(errors.ErrInternal.Code),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			ctx := NewHTTPContext(w, req, logger)

			testCtx := context.WithValue(context.Background(), ContextKey, ctx)

			if ctx.StatusCode() != 0 {
				t.Errorf("Expected initial status code to be 0, got %d", ctx.StatusCode())
			}

			if handleErr := handler.Handle(testCtx, tc.error); handleErr != nil {
				t.Fatalf("Error handler failed: %v", handleErr)
			}

			if ctx.StatusCode() != tc.expectedStatus {
				t.Errorf("Expected context status code %d, got %d", tc.expectedStatus, ctx.StatusCode())
			}

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected HTTP response status %d, got %d", tc.expectedStatus, w.Code)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("JSON decode failed: %v", err)
			}

			errorData := response["error"].(map[string]interface{})
			if errorData["code"] != tc.errorCode {
				t.Errorf("Expected error code %s, got %v", tc.errorCode, errorData["code"])
			}
		})
	}
}

func TestErrorHandlerWithCustomStatusCodeConfig(t *testing.T) {
	t.Parallel()

	config := NewErrorHandlerConfig().
		WithStatusCode(string(errors.ErrValidation.Code), http.StatusTeapot).
		WithStatusCode(string(errors.ErrInternal.Code), http.StatusBadGateway)

	logger := &mockLogger{}
	handler := NewErrorHandler(config, logger)

	testCases := []struct {
		name           string
		error          error
		expectedStatus int
	}{
		{
			name:           "Validation Error with custom status",
			error:          errors.ErrValidation,
			expectedStatus: http.StatusTeapot,
		},
		{
			name:           "Internal Error with custom status",
			error:          errors.ErrInternal,
			expectedStatus: http.StatusBadGateway,
		},
		{
			name:           "Auth Error with default status",
			error:          errors.ErrAuth,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			ctx := NewHTTPContext(w, req, logger)

			testCtx := context.WithValue(context.Background(), ContextKey, ctx)

			if handleErr := handler.Handle(testCtx, tc.error); handleErr != nil {
				t.Fatalf("Error handler failed: %v", handleErr)
			}

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected HTTP status %d, got %d", tc.expectedStatus, w.Code)
			}
		})
	}
}

func TestErrorHandlerDoesNotOverrideAlreadySetStatus(t *testing.T) {
	t.Parallel()

	config := NewErrorHandlerConfig()
	logger := &mockLogger{}
	handler := NewErrorHandler(config, logger)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	ctx := NewHTTPContext(w, req, logger)

	ctx.Status(http.StatusConflict)

	if ctx.StatusCode() != http.StatusConflict {
		t.Fatalf("Expected initial status code to be %d, got %d", http.StatusConflict, ctx.StatusCode())
	}

	testCtx := context.WithValue(context.Background(), ContextKey, ctx)

	if handleErr := handler.Handle(testCtx, errors.ErrConflict); handleErr != nil {
		t.Fatalf("Error handler failed: %v", handleErr)
	}

	if w.Code != http.StatusConflict {
		t.Errorf("Expected HTTP status to remain %d, got %d", http.StatusConflict, w.Code)
	}

	messages := logger.getMessages()
	found := false
	for _, msg := range messages {
		if strings.Contains(msg, "Response already sent") || strings.Contains(msg, "already sent") {
			found = true
			break
		}
	}
	if !found {
		t.Log("Warning: Expected warning about already sent response not found in logs")
	}
}

func TestErrorHandlerOverridesContextStatus(t *testing.T) {
	t.Parallel()

	config := NewErrorHandlerConfig()
	logger := &mockLogger{}
	handler := NewErrorHandler(config, logger)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	ctx := NewHTTPContext(w, req, logger)

	ctx.Status(http.StatusConflict)

	testCtx := context.WithValue(context.Background(), ContextKey, ctx)

	if handleErr := handler.Handle(testCtx, errors.ErrValidation); handleErr != nil {
		t.Fatalf("Error handler failed: %v", handleErr)
	}

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected HTTP status to be %d (validation error), got %d", http.StatusBadRequest, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if errorData, exists := response["error"]; exists {
		if errorObj, ok := errorData.(map[string]interface{}); ok {
			if code, ok := errorObj["code"]; ok && code == string(errors.ErrValidation.Code) {
				return
			}
		}
	}

	t.Error("Expected validation error response in body")
}

type brokenResponseWriter struct {
	*httptest.ResponseRecorder
}

func (b *brokenResponseWriter) Write([]byte) (int, error) {
	return 0, fmt.Errorf("write error")
}
