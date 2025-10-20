package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewHTTPContext(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/test?param=value", nil)
	req.Header.Set("User-Agent", "test-agent")
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)

	if ctx.Method() != "GET" {
		t.Errorf("Expected method GET, got %s", ctx.Method())
	}
	if ctx.Path() != "/test" {
		t.Errorf("Expected path /test, got %s", ctx.Path())
	}
	if ctx.Query("param") != queryParamValue {
		t.Errorf("Expected param value, got %s", ctx.Query("param"))
	}
	if ctx.QueryDefault("missing", "default") != "default" {
		t.Errorf("Expected default value, got %s", ctx.QueryDefault("missing", "default"))
	}
	if ctx.RequestHeader("User-Agent") != "test-agent" {
		t.Errorf("Expected User-Agent test-agent, got %s", ctx.RequestHeader("User-Agent"))
	}
}

func TestHTTPContextJSON(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)

	data := map[string]string{"message": "hello"}
	if err := ctx.Status(201).JSON(data); err != nil {
		t.Fatalf("JSON response failed: %v", err)
	}

	if w.Code != 201 {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	if contentType := w.Header().Get("Content-Type"); contentType != contentTypeJSON {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("JSON decode failed: %v", err)
	}

	if response["message"] != "hello" {
		t.Error("JSON response not correct")
	}
}

func TestHTTPContextString(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)

	if err := ctx.Status(200).String("Hello World"); err != nil {
		t.Fatalf("String response failed: %v", err)
	}

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if body := w.Body.String(); body != "Hello World" {
		t.Errorf("Expected 'Hello World', got %s", body)
	}
}

func TestHTTPContextRedirect(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)

	if err := ctx.Redirect(302, exampleURL); err != nil {
		t.Fatalf("Redirect failed: %v", err)
	}

	if w.Code != 302 {
		t.Errorf("Expected status 302, got %d", w.Code)
	}

	if location := w.Header().Get("Location"); location != exampleURL {
		t.Errorf("Expected Location %s, got %s", exampleURL, location)
	}
}

func TestHTTPContextNoContent(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("DELETE", "/test", nil)
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)

	if err := ctx.NoContent(); err != nil {
		t.Fatalf("NoContent failed: %v", err)
	}

	if w.Code != 204 {
		t.Errorf("Expected status 204, got %d", w.Code)
	}

	if w.Body.Len() != 0 {
		t.Error("Expected empty body")
	}
}

func TestHTTPContextDuplicateResponse(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)

	if err := ctx.JSON(map[string]string{"first": "response"}); err != nil {
		t.Fatalf("First response failed: %v", err)
	}

	if err := ctx.JSON(map[string]string{"second": "response"}); !errors.Is(err, ErrResponseAlreadySent) {
		t.Errorf("Expected ErrResponseAlreadySent, got %v", err)
	}
}

func TestHTTPContextBody(t *testing.T) {
	t.Parallel()

	bodyContent := `{"key": "value"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(bodyContent))
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)

	body, err := ctx.Body()
	if err != nil {
		t.Fatalf("Body read failed: %v", err)
	}

	if string(body) != bodyContent {
		t.Errorf("Expected body %s, got %s", bodyContent, string(body))
	}

	body2, err := ctx.Body()
	if err != nil {
		t.Fatalf("Second body read failed: %v", err)
	}

	if string(body2) != bodyContent {
		t.Error("Second body read should return cached content")
	}
}

func TestHTTPContextWithRequestID(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "custom-request-id")
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)

	if ctx.RequestID() != "custom-request-id" {
		t.Errorf("Expected custom request ID, got %s", ctx.RequestID())
	}
}

func TestHTTPContextSetAndGetParams(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)

	ctx.Set("user_id", 123)
	ctx.Set("role", "admin")

	if value, exists := ctx.Get("user_id"); !exists || value != 123 {
		t.Error("Failed to get user_id parameter")
	}

	if value, exists := ctx.Get("role"); !exists || value != "admin" {
		t.Error("Failed to get role parameter")
	}

	if _, exists := ctx.Get("nonexistent"); exists {
		t.Error("Should not find nonexistent parameter")
	}
}

func TestHTTPContextParamFromString(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)

	ctx.Set("id", "123")
	ctx.Set("number", 456)

	if param := ctx.Param("id"); param != "123" {
		t.Errorf("Expected string param '123', got %s", param)
	}

	if param := ctx.Param("number"); param != "" {
		t.Errorf("Expected empty string for non-string param, got %s", param)
	}

	if param := ctx.Param("missing"); param != "" {
		t.Errorf("Expected empty string for missing param, got %s", param)
	}
}

func TestHTTPContextQueryAll(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/test?param1=value1&param2=value2&param1=value3", nil)
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)

	queryAll := ctx.QueryAll()

	if len(queryAll["param1"]) != 2 {
		t.Error("Expected param1 to have 2 values")
	}

	if queryAll["param1"][0] != "value1" || queryAll["param1"][1] != "value3" {
		t.Error("param1 values not correct")
	}

	if len(queryAll["param2"]) != 1 || queryAll["param2"][0] != "value2" {
		t.Error("param2 value not correct")
	}
}

func TestHTTPContextStartTime(t *testing.T) {
	t.Parallel()

	start := time.Now()
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)

	if ctx.StartTime().Before(start) {
		t.Error("Start time should be after test start")
	}

	if time.Since(ctx.StartTime()) > time.Second {
		t.Error("Start time should be recent")
	}
}

func TestHTTPContextSetContext(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	originalRequestID := ctx.RequestID()

	newCtx := context.WithValue(context.Background(), customKey, "custom_value")
	ctx.WithParentContext(newCtx)

	if ctx.ParentContext().Value(customKey) != "custom_value" {
		t.Error("Custom context value not set")
	}

	if ctx.RequestID() != originalRequestID {
		t.Error("Request ID should be preserved when setting context")
	}
}

func TestHTTPContextDataResponse(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)

	data := []byte("binary data")
	if err := ctx.Data(appOctetStream, data); err != nil {
		t.Fatalf("Data response failed: %v", err)
	}

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if contentType := w.Header().Get("Content-Type"); contentType != appOctetStream {
		t.Errorf("Expected Content-Type application/octet-stream, got %s", contentType)
	}

	if !strings.Contains(w.Body.String(), "binary data") {
		t.Error("Response body not correct")
	}
}

func TestHTTPContextResponseAlreadySent(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)

	if err := ctx.JSON(map[string]string{"first": "response"}); err != nil {
		t.Fatalf("First response failed: %v", err)
	}

	tests := []struct {
		name string
		fn   func() error
	}{
		{"String", func() error { return ctx.String("test") }},
		{"Data", func() error { return ctx.Data("text/plain", []byte("test")) }},
		{"Redirect", func() error { return ctx.Redirect(302, "/") }},
		{"NoContent", func() error { return ctx.NoContent() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); !errors.Is(err, ErrResponseAlreadySent) {
				t.Errorf("Expected ErrResponseAlreadySent for %s, got %v", tt.name, err)
			}
		})
	}
}

func TestHTTPContextJSONMarshalError(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)

	invalidData := make(chan int)
	err := ctx.JSON(invalidData)

	if err == nil {
		t.Error("Expected JSON marshal error for channel type")
	}
}

func TestHTTPContextFileUpload(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	upload := ctx.FileUpload()

	if upload == nil {
		t.Error("FileUpload should not return nil")
	}
}

func TestHTTPContextStreaming(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	stream := ctx.Streaming()

	if stream == nil {
		t.Error("Streaming should not return nil")
	}
}

func TestHTTPContextWebsocket(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	ws := ctx.Websocket()

	if ws == nil {
		t.Error("Websocket should not return nil")
	}
}

func TestHTTPContextBodyReadError(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("POST", "/test", &errorReader{})
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)

	_, err := ctx.Body()
	if err == nil {
		t.Error("Expected error reading body")
	}
}

type errorReader struct{}

func (e *errorReader) Read(_ []byte) (n int, err error) {
	return 0, errors.New("read error")
}
