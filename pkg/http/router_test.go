package http

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestRouterHTTPMethods(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	router := NewRouter(logger)

	methods := []string{"POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	for _, method := range methods {
		handlerCalled := false
		handler := func(ctx contracts.HTTPContext) error {
			handlerCalled = true
			return ctx.JSON(map[string]string{"method": ctx.Method()})
		}

		switch method {
		case "POST":
			router.POST("/test", handler)
		case "PUT":
			router.PUT("/test", handler)
		case "DELETE":
			router.DELETE("/test", handler)
		case "PATCH":
			router.PATCH("/test", handler)
		case "HEAD":
			router.HEAD("/test", handler)
		case "OPTIONS":
			router.OPTIONS("/test", handler)
		}

		req := httptest.NewRequest(method, "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if !handlerCalled {
			t.Errorf("%s handler was not called", method)
		}
	}
}

func TestRouterGroupNested(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	router := NewRouter(logger)

	handlerCalled := false
	handler := func(ctx contracts.HTTPContext) error {
		handlerCalled = true
		return ctx.JSON(map[string]string{"path": ctx.Path()})
	}

	api := router.Group("/api")
	v1 := api.Group("/v1")
	users := v1.Group("/users")
	users.GET("/profile", handler)

	req := httptest.NewRequest("GET", "/api/v1/users/profile", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("Nested group handler was not called")
	}
}

func TestRouterGroupMiddleware(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	router := NewRouter(logger)

	middlewareCalled := false
	middleware := func(next contracts.HTTPHandler) contracts.HTTPHandler {
		return func(ctx contracts.HTTPContext) error {
			middlewareCalled = true
			ctx.Set("group", "middleware")
			return next(ctx)
		}
	}

	handlerCalled := false
	handler := func(ctx contracts.HTTPContext) error {
		handlerCalled = true
		value, _ := ctx.Get("group")
		return ctx.JSON(map[string]interface{}{"middleware": value})
	}

	api := router.Group("/api", middleware)
	api.GET("/test", handler)

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !middlewareCalled {
		t.Error("Group middleware was not called")
	}
	if !handlerCalled {
		t.Error("Handler was not called")
	}
}

func TestRouterStaticFileNotFound(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	logger := &mockLogger{}
	router := NewRouter(logger)
	router.Static("/static", tmpDir)

	req := httptest.NewRequest("GET", "/static/nonexistent.txt", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 500 {
		t.Errorf("Expected status 500 for missing file, got %d", w.Code)
	}
}

func TestRouterStaticDirectory(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	indexFile := filepath.Join(subDir, "index.html")
	content := "<html><body>Index</body></html>"
	if err := os.WriteFile(indexFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create index file: %v", err)
	}

	logger := &mockLogger{}
	router := NewRouter(logger)
	router.Static("/static", tmpDir)

	req := httptest.NewRequest("GET", "/static/subdir", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Index") {
		t.Error("Expected index.html content")
	}
}

func TestRouterStaticDirectoryNoIndex(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	logger := &mockLogger{}
	router := NewRouter(logger)
	router.Static("/static", tmpDir)

	req := httptest.NewRequest("GET", "/static/subdir", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 500 {
		t.Errorf("Expected status 500 for directory without index, got %d", w.Code)
	}
}

func TestRouterStaticFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Hello, Static!"

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	logger := &mockLogger{}
	router := NewRouter(logger)
	router.StaticFile("/file", testFile)

	req := httptest.NewRequest("GET", "/file", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if w.Body.String() != content {
		t.Errorf("Expected content %s, got %s", content, w.Body.String())
	}
}

func TestRouterHandleNilHandler(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	router := NewRouter(logger)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for nil handler")
		}
	}()

	router.Handle("GET", "/test", nil)
}

func TestRouterMatchPatternEdgeCases(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	router := NewRouter(logger).(*httpRouter)

	tests := []struct {
		pattern string
		path    string
		match   bool
	}{
		{"/users/:id/posts/:postId", "/users/123/posts/456", true},
		{"/users/:id/posts/:postId", "/users/123/posts", false},
		{"/files/*", "/files", true},
		{"/files/*", "/files/", true},
		{"/files/*", "/files/path/to/file", true},
		{"/:category/:id", "/books/123", true},
		{"/:category/:id", "/books", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"->"+tt.path, func(t *testing.T) {
			params := router.matchPattern(tt.pattern, tt.path)
			matched := params != nil

			if matched != tt.match {
				t.Errorf("Expected match=%v for pattern %s and path %s, got %v",
					tt.match, tt.pattern, tt.path, matched)
			}
		})
	}
}

func TestRouterStaticUnsafePaths(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	logger := &mockLogger{}
	router := NewRouter(logger)
	router.Static("/static", tmpDir)

	unsafePaths := []struct {
		name string
		path string
	}{
		{"Parent directory traversal", "/static/../../../etc/passwd"},
		{"URL encoded traversal", "/static/..%2F..%2F..%2Fetc%2Fpasswd"},
	}

	for _, test := range unsafePaths {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest("GET", test.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code == 200 {
				t.Errorf("Unsafe path %s should be rejected", test.path)
			}
		})
	}

	t.Run("Null byte in path", func(t *testing.T) {
		t.Parallel()
		logger := &mockLogger{}
		router := NewRouter(logger)
		router.Static("/static", tmpDir)
		req := httptest.NewRequest("GET", "/static/safe", nil)
		w := httptest.NewRecorder()
		ctx := NewHTTPContext(w, req, logger)
		ctx.Set("*", "\x00malicious")
		if isPathSafe(tmpDir, "\x00malicious") {
			t.Error("Path with null byte should be rejected")
		}
	})
}

func TestIsPathSafeEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		root string
		path string
		want bool
	}{
		{"Symlink attack", "/tmp", "link", true},
		{"Hidden file", "/tmp", ".hidden", true},
		{"Multiple slashes", "/tmp", "dir//file", true},
		{"Trailing slash", "/tmp", "dir/", true},
		{"Root traversal", "", "../file", false},
		{"Complex traversal", "/var/www", "uploads/../../etc/passwd", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPathSafe(tt.root, tt.path); got != tt.want {
				t.Errorf("isPathSafe(%q, %q) = %v, want %v", tt.root, tt.path, got, tt.want)
			}
		})
	}
}
