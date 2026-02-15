package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecovery_NoPanic(t *testing.T) {
	t.Parallel()
	handler := Recovery(nil)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestRecovery_CatchesPanic(t *testing.T) {
	t.Parallel()
	handler := Recovery(nil)(
		http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			panic("boom")
		}),
	)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "internal error") {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
}

func TestRecovery_LogsPanic(t *testing.T) {
	t.Parallel()
	var logged bool
	logFn := func(_ string, _ ...any) { logged = true }
	handler := Recovery(logFn)(
		http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			panic("test panic")
		}),
	)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	if !logged {
		t.Fatal("expected panic to be logged")
	}
}

func TestRecovery_ContentType(t *testing.T) {
	t.Parallel()
	handler := Recovery(nil)(
		http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			panic("test")
		}),
	)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	ct := rr.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Fatalf("expected application/json, got %q", ct)
	}
}
