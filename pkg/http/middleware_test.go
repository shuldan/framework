package http

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/shuldan/framework/pkg/contracts"
	"github.com/shuldan/framework/pkg/errors"
)

func TestMiddlewares(t *testing.T) {
	t.Parallel()
	logger := &mockLogger{}
	t.Run("LoggingMiddleware", func(t *testing.T) {
		middleware := LoggingMiddleware(logger)
		handler := func(ctx contracts.HTTPContext) error {
			return ctx.JSON(map[string]string{"status": "ok"})
		}
		wrappedHandler := middleware(handler)
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		ctx := NewHTTPContext(w, req, logger)
		if err := wrappedHandler(ctx); err != nil {
			t.Fatalf("Handler failed: %v", err)
		}
		messages := logger.getMessages()
		if len(messages) == 0 {
			t.Error("Expected logging messages")
		}
	})

	t.Run("RecoveryMiddleware", func(t *testing.T) {
		middleware := RecoveryMiddleware(logger)
		handler := func(ctx contracts.HTTPContext) error {
			panic("test panic")
		}
		wrappedHandler := middleware(handler)
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		ctx := NewHTTPContext(w, req, logger)
		err := wrappedHandler(ctx)
		if err == nil {
			t.Error("Expected panic to be recovered as error")
		}
	})

	t.Run("CORSMiddleware", func(t *testing.T) {
		config := CORSConfig{
			AllowOrigins: []string{"https://example.com"},
		}
		middleware := CORSMiddleware(config)
		handler := func(ctx contracts.HTTPContext) error {
			return ctx.JSON(map[string]string{"status": "ok"})
		}
		wrappedHandler := middleware(handler)
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", exampleURL)
		w := httptest.NewRecorder()
		ctx := NewHTTPContext(w, req, logger)
		if err := wrappedHandler(ctx); err != nil {
			t.Fatalf("Handler failed: %v", err)
		}
		if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != exampleURL {
			t.Errorf("Expected CORS origin https://example.com, got %s", origin)
		}
	})
}

func TestLoggingMiddlewareWithUserData(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	middleware := LoggingMiddleware(logger)

	handler := func(ctx contracts.HTTPContext) error {
		ctx.Set("user_id", "12345")
		return ctx.JSON(map[string]string{"status": "ok"})
	}

	wrappedHandler := middleware(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "TestAgent/1.0")
	req.Header.Set("X-Forwarded-For", "192.168.1.1, 10.0.0.1")
	w := httptest.NewRecorder()

	ctx := NewHTTPContext(w, req, logger)
	if err := wrappedHandler(ctx); err != nil {
		t.Fatalf("Handler failed: %v", err)
	}

	messages := logger.getMessages()
	found := false
	for _, msg := range messages {
		if strings.Contains(msg, "user_id=12345") &&
			strings.Contains(msg, "user_agent=TestAgent/1.0") &&
			strings.Contains(msg, "client_ip=192.168.1.1") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected log message with user data")
	}
}

func TestLoggingMiddlewareErrorLevels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		statusCode    int
		expectedLevel string
	}{
		{"Client Error", 404, "WARN"},
		{"Server Error", 500, "ERROR"},
		{"Success", 200, "INFO"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &mockLogger{}
			middleware := LoggingMiddleware(logger)

			handler := func(ctx contracts.HTTPContext) error {
				ctx.Status(tt.statusCode)
				return ctx.JSON(map[string]string{"status": "test"})
			}

			wrappedHandler := middleware(handler)
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()

			ctx := NewHTTPContext(w, req, logger)
			if err := wrappedHandler(ctx); err != nil {
				t.Fatalf("Handler failed: %v", err)
			}

			messages := logger.getMessages()
			found := false
			for _, msg := range messages {
				if strings.Contains(msg, "["+tt.expectedLevel+"]") {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected %s level log message", tt.expectedLevel)
			}
		})
	}
}

func TestRecoveryMiddlewareNilLogger(t *testing.T) {
	t.Parallel()

	middleware := RecoveryMiddleware(nil)
	handler := func(ctx contracts.HTTPContext) error {
		panic("test panic")
	}

	wrappedHandler := middleware(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	ctx := NewHTTPContext(w, req, nil)
	err := wrappedHandler(ctx)

	if err == nil {
		t.Error("Expected panic to be recovered as error")
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	t.Parallel()

	middleware := RequestIDMiddleware()
	handler := func(ctx contracts.HTTPContext) error {
		return ctx.JSON(map[string]string{"request_id": ctx.RequestID()})
	}

	wrappedHandler := middleware(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	if err := wrappedHandler(ctx); err != nil {
		t.Fatalf("Handler failed: %v", err)
	}

	if requestID := w.Header().Get("X-Request-ID"); requestID == "" {
		t.Error("Expected X-Request-ID header to be set")
	}
}

func TestErrorHandlerMiddleware(t *testing.T) {
	t.Parallel()

	config := NewErrorHandlerConfig()
	logger := &mockLogger{}
	errorHandler := NewErrorHandler(config, logger)
	middleware := ErrorHandlerMiddleware(errorHandler)

	handler := func(ctx contracts.HTTPContext) error {
		return errors.ErrInternal
	}

	wrappedHandler := middleware(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	ctx := NewHTTPContext(w, req, logger)
	if err := wrappedHandler(ctx); err != nil {
		t.Fatalf("Error handler middleware should handle errors: %v", err)
	}
}

func TestCORSMiddlewareNoOrigin(t *testing.T) {
	t.Parallel()

	config := CORSConfig{}
	middleware := CORSMiddleware(config)
	handler := func(ctx contracts.HTTPContext) error {
		return ctx.JSON(map[string]string{"status": "ok"})
	}

	wrappedHandler := middleware(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	if err := wrappedHandler(ctx); err != nil {
		t.Fatalf("Handler failed: %v", err)
	}

	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "" {
		t.Error("Expected no CORS headers when no origin")
	}
}

func TestCORSMiddlewareForbiddenOrigin(t *testing.T) {
	t.Parallel()

	config := CORSConfig{
		AllowOrigins: []string{"https://allowed.com"},
	}
	middleware := CORSMiddleware(config)
	handler := func(ctx contracts.HTTPContext) error {
		return ctx.JSON(map[string]string{"status": "ok"})
	}

	wrappedHandler := middleware(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://forbidden.com")
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	if err := wrappedHandler(ctx); err != nil {
		t.Fatalf("Handler failed: %v", err)
	}

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}
}

func TestCORSMiddlewareWildcardOrigin(t *testing.T) {
	t.Parallel()

	config := CORSConfig{
		AllowOrigins: []string{"*"},
	}
	middleware := CORSMiddleware(config)
	handler := func(ctx contracts.HTTPContext) error {
		return ctx.JSON(map[string]string{"status": "ok"})
	}

	wrappedHandler := middleware(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	if err := wrappedHandler(ctx); err != nil {
		t.Fatalf("Handler failed: %v", err)
	}

	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("Expected wildcard origin *, got %s", origin)
	}
}

func TestCORSMiddlewareWildcardWithCredentials(t *testing.T) {
	t.Parallel()

	config := CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowCredentials: true,
	}
	middleware := CORSMiddleware(config)
	handler := func(ctx contracts.HTTPContext) error {
		return ctx.JSON(map[string]string{"status": "ok"})
	}

	wrappedHandler := middleware(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	if err := wrappedHandler(ctx); err != nil {
		t.Fatalf("Handler failed: %v", err)
	}

	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "https://example.com" {
		t.Errorf("Expected specific origin with credentials, got %s", origin)
	}

	if credentials := w.Header().Get("Access-Control-Allow-Credentials"); credentials != "true" {
		t.Error("Expected Allow-Credentials header")
	}
}

func TestCORSMiddlewareSubdomainWildcard(t *testing.T) {
	t.Parallel()

	config := CORSConfig{
		AllowOrigins: []string{"*.example.com"},
	}
	middleware := CORSMiddleware(config)
	handler := func(ctx contracts.HTTPContext) error {
		return ctx.JSON(map[string]string{"status": "ok"})
	}

	tests := []struct {
		origin   string
		expected string
		allowed  bool
	}{
		{"https://api.example.com", "https://api.example.com", true},
		{"https://example.com", "https://example.com", true},
		{"http://example.com", "http://example.com", true},
		{"https://sub.api.example.com", "https://sub.api.example.com", true},
		{"https://notexample.com", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.origin, func(t *testing.T) {
			wrappedHandler := middleware(handler)
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Origin", tt.origin)
			w := httptest.NewRecorder()
			logger := &mockLogger{}

			ctx := NewHTTPContext(w, req, logger)
			if err := wrappedHandler(ctx); err != nil {
				t.Fatalf("Handler failed: %v", err)
			}

			origin := w.Header().Get("Access-Control-Allow-Origin")
			if tt.allowed && origin != tt.expected {
				t.Errorf("Expected origin %s, got %s", tt.expected, origin)
			}
			if !tt.allowed && w.Code != http.StatusForbidden {
				t.Errorf("Expected forbidden status for %s", tt.origin)
			}
		})
	}
}

func TestCORSMiddlewareOptionsRequest(t *testing.T) {
	t.Parallel()

	config := CORSConfig{
		AllowOrigins: []string{"https://example.com"},
		MaxAge:       3600 * time.Second,
	}
	middleware := CORSMiddleware(config)
	handler := func(ctx contracts.HTTPContext) error {
		return ctx.JSON(map[string]string{"status": "ok"})
	}

	wrappedHandler := middleware(handler)
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	if err := wrappedHandler(ctx); err != nil {
		t.Fatalf("Handler failed: %v", err)
	}

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204 for OPTIONS, got %d", w.Code)
	}

	if methods := w.Header().Get("Access-Control-Allow-Methods"); methods == "" {
		t.Error("Expected Access-Control-Allow-Methods header")
	}

	if headers := w.Header().Get("Access-Control-Allow-Headers"); headers == "" {
		t.Error("Expected Access-Control-Allow-Headers header")
	}

	if maxAge := w.Header().Get("Access-Control-Max-Age"); maxAge != "3600" {
		t.Errorf("Expected Max-Age 3600, got %s", maxAge)
	}
}

func TestCheckOriginInvalidURL(t *testing.T) {
	t.Parallel()

	config := CORSConfig{AllowOrigins: []string{"*"}}
	_, valid := checkOrigin(config, "invalid-url")

	if valid {
		t.Error("Expected invalid URL to be rejected")
	}
}

func TestSecurityMiddleware(t *testing.T) {
	t.Parallel()

	middleware := SecurityHeadersMiddleware(
		securityHeadersConfig{
			enabled:        true,
			csp:            "default-src 'self'; script-src 'self' 'unsafe-inline';",
			xFrameOptions:  "DENY",
			xXSSProtection: "1; mode=block",
			referrerPolicy: "strict-origin-when-cross-origin",
		},
	)
	handler := func(ctx contracts.HTTPContext) error {
		return ctx.JSON(map[string]string{"status": "ok"})
	}

	wrappedHandler := middleware(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	if err := wrappedHandler(ctx); err != nil {
		t.Fatalf("Handler failed: %v", err)
	}

	expectedHeaders := map[string]string{
		"Content-Security-Policy": "default-src 'self'; script-src 'self' 'unsafe-inline';",
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
		"X-XSS-Protection":        "1; mode=block",
		"Referrer-Policy":         "strict-origin-when-cross-origin",
		"Server":                  "",
	}

	for header, expected := range expectedHeaders {
		if actual := w.Header().Get(header); actual != expected {
			t.Errorf("Expected %s: %s, got %s", header, expected, actual)
		}
	}

	if csp := w.Header().Get("Content-Security-Policy"); csp == "" {
		t.Error("Expected Content-Security-Policy header")
	}
}

func TestHSTSMiddleware(t *testing.T) {
	t.Parallel()

	middleware := HSTSMiddleware(
		hstsConfig{
			enabled:           true,
			maxAge:            31536000 * time.Second,
			includeSubdomains: true,
			preload:           true,
		},
	)
	handler := func(ctx contracts.HTTPContext) error {
		return ctx.JSON(map[string]string{"status": "ok"})
	}

	wrappedHandler := middleware(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	req.TLS = &tls.ConnectionState{}
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	if err := wrappedHandler(ctx); err != nil {
		t.Fatalf("Handler failed: %v", err)
	}

	hsts := w.Header().Get("Strict-Transport-Security")
	if !strings.Contains(hsts, "max-age=31536000") {
		t.Error("Expected max-age in HSTS header")
	}
	if !strings.Contains(hsts, "includeSubDomains") {
		t.Error("Expected includeSubDomains in HSTS header")
	}
	if !strings.Contains(hsts, "preload") {
		t.Error("Expected preload in HSTS header")
	}
}

func TestHSTSMiddlewareNoTLS(t *testing.T) {
	t.Parallel()

	middleware := HSTSMiddleware(
		hstsConfig{
			enabled:           true,
			maxAge:            31536000 * time.Second,
			includeSubdomains: true,
			preload:           true,
		},
	)
	handler := func(ctx contracts.HTTPContext) error {
		return ctx.JSON(map[string]string{"status": "ok"})
	}

	wrappedHandler := middleware(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	if err := wrappedHandler(ctx); err != nil {
		t.Fatalf("Handler failed: %v", err)
	}

	if hsts := w.Header().Get("Strict-Transport-Security"); hsts != "" {
		t.Error("Expected no HSTS header without TLS")
	}
}
