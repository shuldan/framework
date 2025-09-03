package http

import (
	"net/http/httptest"
	"testing"

	"github.com/shuldan/framework/pkg/contracts"
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
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()
		ctx := NewHTTPContext(w, req, logger)
		if err := wrappedHandler(ctx); err != nil {
			t.Fatalf("Handler failed: %v", err)
		}
		if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "https://example.com" {
			t.Errorf("Expected CORS origin https://example.com, got %s", origin)
		}
	})
}
