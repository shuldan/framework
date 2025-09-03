package http

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
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

	if err := ctx.Redirect(302, "https://example.com"); err != nil {
		t.Fatalf("Redirect failed: %v", err)
	}

	if w.Code != 302 {
		t.Errorf("Expected status 302, got %d", w.Code)
	}

	if location := w.Header().Get("Location"); location != "https://example.com" {
		t.Errorf("Expected Location https://example.com, got %s", location)
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

	if err := ctx.JSON(map[string]string{"second": "response"}); err != ErrResponseAlreadySent {
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
