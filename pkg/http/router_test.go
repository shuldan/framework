package http

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/shuldan/framework/pkg/contracts"
)

func TestRouter(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	router := NewRouter(logger)

	handlerCalled := false
	handler := func(ctx contracts.HTTPContext) error {
		handlerCalled = true
		return ctx.JSON(map[string]string{"status": "ok"})
	}

	router.GET("/test", handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("Handler was not called")
	}

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRouterParams(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	router := NewRouter(logger)

	var capturedID string
	handler := func(ctx contracts.HTTPContext) error {
		capturedID = ctx.Param("id")
		return ctx.JSON(map[string]string{"id": capturedID})
	}

	router.GET("/users/:id", handler)

	req := httptest.NewRequest("GET", "/users/123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if capturedID != "123" {
		t.Errorf("Expected ID 123, got %s", capturedID)
	}
}

func TestRouterNotFound(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	router := NewRouter(logger)

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != 500 {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestRouterMiddleware(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	router := NewRouter(logger)

	middlewareCalled := false
	middleware := func(next contracts.HTTPHandler) contracts.HTTPHandler {
		return func(ctx contracts.HTTPContext) error {
			middlewareCalled = true
			ctx.Set("middleware", "called")
			return next(ctx)
		}
	}

	handlerCalled := false
	handler := func(ctx contracts.HTTPContext) error {
		handlerCalled = true
		value, _ := ctx.Get("middleware")
		return ctx.JSON(map[string]interface{}{"middleware": value})
	}

	router.Use(middleware)
	router.GET("/test", handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if !middlewareCalled {
		t.Error("Middleware was not called")
	}

	if !handlerCalled {
		t.Error("Handler was not called")
	}
}

func TestRouterGroup(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	router := NewRouter(logger)

	handlerCalled := false
	handler := func(ctx contracts.HTTPContext) error {
		handlerCalled = true
		return ctx.JSON(map[string]string{"path": ctx.Path()})
	}

	api := router.Group("/api/v1")
	api.GET("/users", handler)

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("Handler was not called")
	}

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRouterStatic(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Hello, World!"

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	logger := &mockLogger{}
	router := NewRouter(logger)
	router.Static("/static", tmpDir)

	req := httptest.NewRequest("GET", "/static/test.txt", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() != content {
		t.Errorf("Expected content %s, got %s", content, w.Body.String())
	}
}

func TestRouterWildcard(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	router := NewRouter(logger)

	var capturedPath string
	handler := func(ctx contracts.HTTPContext) error {
		capturedPath = ctx.Param("*")
		return ctx.JSON(map[string]string{"path": capturedPath})
	}

	router.GET("/files/*", handler)

	req := httptest.NewRequest("GET", "/files/path/to/file.txt", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if capturedPath != "path/to/file.txt" {
		t.Errorf("Expected wildcard 'path/to/file.txt', got %s", capturedPath)
	}
}

func TestRouterMethodNotAllowed(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	router := NewRouter(logger)

	handler := func(ctx contracts.HTTPContext) error {
		return ctx.JSON(map[string]string{"status": "ok"})
	}

	router.GET("/test", handler)

	req := httptest.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != 500 {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestIsPathSafe(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		root string
		path string
		want bool
	}{
		{"Normal path", "/tmp", "file.txt", true},
		{"Absolute path", "/tmp", "/etc/passwd", false},
		{"Parent directory", "/tmp", "../etc/passwd", false},
		{"Nested parent", "/tmp", "dir/../../etc/passwd", false},
		{"Current directory", "/tmp", "./file.txt", true},
		{"Empty path", "/tmp", "", true},
		{"Just parent", "/tmp", "..", false},
		{"No root", "", "file.txt", true},
		{"No root with parent", "", "../file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPathSafe(tt.root, tt.path); got != tt.want {
				t.Errorf("isPathSafe(%q, %q) = %v, want %v", tt.root, tt.path, got, tt.want)
			}
		})
	}
}
