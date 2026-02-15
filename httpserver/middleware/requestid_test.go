package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestID_Generates(t *testing.T) {
	t.Parallel()
	var ctxID string
	handler := RequestID()(
		http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			ctxID = IDFromContext(r.Context())
		}),
	)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	respID := rr.Header().Get(HeaderRequestID)
	if respID == "" {
		t.Fatal("expected X-Request-Id header")
	}
	if ctxID != respID {
		t.Fatalf("context ID %q != header ID %q", ctxID, respID)
	}
}

func TestRequestID_ReusesExisting(t *testing.T) {
	t.Parallel()
	handler := RequestID()(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set(HeaderRequestID, "existing-id")
	handler.ServeHTTP(rr, req)
	if id := rr.Header().Get(HeaderRequestID); id != "existing-id" {
		t.Fatalf("expected 'existing-id', got %q", id)
	}
}

func TestIDFromContext_Empty(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest("GET", "/", nil)
	if id := IDFromContext(req.Context()); id != "" {
		t.Fatalf("expected empty ID, got %q", id)
	}
}

func TestRequestID_GeneratedFormat(t *testing.T) {
	t.Parallel()
	var ctxID string
	handler := RequestID()(
		http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			ctxID = IDFromContext(r.Context())
		}),
	)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	if len(ctxID) == 0 {
		t.Fatal("expected non-empty request ID")
	}
	_ = rr
}
