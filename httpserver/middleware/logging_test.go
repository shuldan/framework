package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLogging_LogsRequest(t *testing.T) {
	t.Parallel()
	var args []any
	logFn := func(_ string, a ...any) { args = a }
	handler := Logging(logFn)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
		}),
	)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest("POST", "/users", nil))
	found := findKV(args, "method")
	if found != "POST" {
		t.Fatalf("expected method 'POST', got %v", found)
	}
	found = findKV(args, "status")
	if found != http.StatusCreated {
		t.Fatalf("expected status 201, got %v", found)
	}
}

func TestLogging_DefaultStatus(t *testing.T) {
	t.Parallel()
	var args []any
	logFn := func(_ string, a ...any) { args = a }
	handler := Logging(logFn)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("ok"))
		}),
	)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	found := findKV(args, "status")
	if found != http.StatusOK {
		t.Fatalf("expected status 200, got %v", found)
	}
}

func TestLogging_DoubleWriteHeader(t *testing.T) {
	t.Parallel()
	var args []any
	logFn := func(_ string, a ...any) { args = a }
	handler := Logging(logFn)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusAccepted)
			w.WriteHeader(http.StatusBadRequest)
		}),
	)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	found := findKV(args, "status")
	if found != http.StatusAccepted {
		t.Fatalf("expected status 202, got %v", found)
	}
}

func TestStatusWriter_Unwrap(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: rr, status: http.StatusOK}
	if sw.Unwrap() != rr {
		t.Fatal("Unwrap should return inner ResponseWriter")
	}
}

func findKV(args []any, key string) any {
	for i := 0; i+1 < len(args); i += 2 {
		if args[i] == key {
			return args[i+1]
		}
	}
	return nil
}
