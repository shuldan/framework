package http

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStreamingContext(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/stream", nil)
	w := &mockFlushableResponseWriter{ResponseRecorder: httptest.NewRecorder()}
	logger := &mockLogger{}
	ctx := NewHTTPContext(w, req, logger)
	stream := ctx.Streaming()
	stream.SetContentType("text/plain").SetHeader("Cache-Control", "no-cache")
	if err := stream.WriteChunk([]byte("Hello ")); err != nil {
		t.Fatalf("WriteChunk failed: %v", err)
	}
	if err := stream.WriteStringChunk("World!"); err != nil {
		t.Fatalf("WriteStringChunk failed: %v", err)
	}
	stream.Flush()
	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if body := w.Body.String(); body != "Hello World!" {
		t.Errorf("Expected 'Hello World!', got %s", body)
	}
	if contentType := w.Header().Get("Content-Type"); contentType != "text/plain" {
		t.Errorf("Expected Content-Type text/plain, got %s", contentType)
	}
	if ctx.StatusCode() != 200 {
		t.Errorf("Expected status code 200, got %d", ctx.StatusCode())
	}
}

func TestStreamingContextCloseNotify(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/stream", nil)
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	stream := ctx.Streaming()

	closeNotify := stream.CloseNotify()
	if closeNotify == nil {
		t.Error("CloseNotify returned nil channel")
	}

	if stream.IsClientClosed() {
		t.Error("Expected client not to be closed initially")
	}
}

func TestStreamingContextNoFlusher(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/stream", nil)
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	stream := ctx.Streaming()

	if err := stream.WriteChunk([]byte("test")); err != nil {
		t.Fatalf("WriteChunk failed: %v", err)
	}

	stream.Flush()

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestStreamingContextDefaultContentType(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/stream", nil)
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	stream := ctx.Streaming()

	if err := stream.WriteChunk([]byte("test")); err != nil {
		t.Fatalf("WriteChunk failed: %v", err)
	}

	if contentType := w.Header().Get("Content-Type"); contentType != "text/plain" {
		t.Errorf("Expected default Content-Type text/plain, got %s", contentType)
	}
}

func TestStreamingContextClientDisconnect(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/stream", nil)
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	logger := &mockLogger{}

	httpCtx := NewHTTPContext(w, req, logger)
	stream := httpCtx.Streaming()

	cancel()

	closeNotify := stream.CloseNotify()
	select {
	case <-closeNotify:
		t.Log("Client disconnect detected")
	case <-time.After(100 * time.Millisecond):
		t.Log("No disconnect signal received")
	}

	if !stream.IsClientClosed() {
		t.Error("Expected client to be considered closed after context cancellation")
	}
}

func TestStreamingContextMultipleCloseNotify(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/stream", nil)
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	stream := ctx.Streaming()

	closeNotify1 := stream.CloseNotify()
	closeNotify2 := stream.CloseNotify()

	if closeNotify1 != closeNotify2 {
		t.Error("CloseNotify should return the same channel on multiple calls")
	}
}
