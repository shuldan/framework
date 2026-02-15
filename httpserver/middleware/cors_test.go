package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS_AllowsMatchingOrigin(t *testing.T) {
	t.Parallel()
	cfg := CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type"},
		MaxAge:         3600,
	}
	handler := CORS(cfg)(okHandler())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	handler.ServeHTTP(rr, req)
	origin := rr.Header().Get("Access-Control-Allow-Origin")
	if origin != "https://example.com" {
		t.Fatalf("expected origin, got %q", origin)
	}
	if rr.Header().Get("Access-Control-Max-Age") != "3600" {
		t.Errorf("expected max-age 3600")
	}
}

func TestCORS_Wildcard(t *testing.T) {
	t.Parallel()
	cfg := CORSConfig{AllowedOrigins: []string{"*"}}
	handler := CORS(cfg)(okHandler())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://any.com")
	handler.ServeHTTP(rr, req)
	if v := rr.Header().Get("Access-Control-Allow-Origin"); v != "*" {
		t.Fatalf("expected '*', got %q", v)
	}
}

func TestCORS_Preflight(t *testing.T) {
	t.Parallel()
	cfg := CORSConfig{AllowedOrigins: []string{"*"}, AllowedMethods: []string{"POST"}}
	handler := CORS(cfg)(okHandler())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
}

func TestCORS_NoOriginHeader(t *testing.T) {
	t.Parallel()
	cfg := CORSConfig{AllowedOrigins: []string{"*"}}
	handler := CORS(cfg)(okHandler())
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	if v := rr.Header().Get("Access-Control-Allow-Origin"); v != "" {
		t.Fatalf("expected no CORS header, got %q", v)
	}
}

func TestCORS_UnmatchedOrigin(t *testing.T) {
	t.Parallel()
	cfg := CORSConfig{AllowedOrigins: []string{"https://allowed.com"}}
	handler := CORS(cfg)(okHandler())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://other.com")
	handler.ServeHTTP(rr, req)
	if v := rr.Header().Get("Access-Control-Allow-Origin"); v != "" {
		t.Fatalf("expected no CORS header, got %q", v)
	}
}

func TestCORS_MaxAgeZero(t *testing.T) {
	t.Parallel()
	cfg := CORSConfig{
		AllowedOrigins: []string{"*"},
		MaxAge:         0,
	}
	handler := CORS(cfg)(okHandler())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://any.com")
	handler.ServeHTTP(rr, req)
	if v := rr.Header().Get("Access-Control-Max-Age"); v != "" {
		t.Fatalf("expected no Max-Age header, got %q", v)
	}
}

func TestCORS_NoMethodsOrHeaders(t *testing.T) {
	t.Parallel()
	cfg := CORSConfig{AllowedOrigins: []string{"*"}}
	handler := CORS(cfg)(okHandler())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://test.com")
	handler.ServeHTTP(rr, req)
	if v := rr.Header().Get("Access-Control-Allow-Methods"); v != "" {
		t.Errorf("expected no methods header, got %q", v)
	}
	if v := rr.Header().Get("Access-Control-Allow-Headers"); v != "" {
		t.Errorf("expected no headers header, got %q", v)
	}
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
