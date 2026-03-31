package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockLogger struct {
	args []any
}

func (m *mockLogger) Info(_ string, args ...any)  { m.args = args }
func (m *mockLogger) Warn(_ string, args ...any)  { m.args = args }
func (m *mockLogger) Error(_ string, args ...any) { m.args = args }

func TestLogging_LogsRequest(t *testing.T) {
	t.Parallel()
	log := &mockLogger{}
	handler := Logging(log)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
		}),
	)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest("POST", "/users", nil))
	found := findKV(log.args, "method")
	if found != "POST" {
		t.Fatalf("expected method 'POST', got %v", found)
	}
	found = findKV(log.args, "status")
	if found != http.StatusCreated {
		t.Fatalf("expected status 201, got %v", found)
	}
}

func TestLogging_DefaultStatus(t *testing.T) {
	t.Parallel()
	log := &mockLogger{}
	handler := Logging(log)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("ok"))
		}),
	)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	found := findKV(log.args, "status")
	if found != http.StatusOK {
		t.Fatalf("expected status 200, got %v", found)
	}
}

func TestLogging_DoubleWriteHeader(t *testing.T) {
	t.Parallel()
	log := &mockLogger{}
	handler := Logging(log)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusAccepted)
			w.WriteHeader(http.StatusBadRequest)
		}),
	)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	found := findKV(log.args, "status")
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

func TestLogging_WarnOnClientError(t *testing.T) {
	t.Parallel()
	log := &mockLogger{}
	handler := Logging(log)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}),
	)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest("GET", "/missing", nil))
	found := findKV(log.args, "status")
	if found != http.StatusNotFound {
		t.Fatalf("expected status 404, got %v", found)
	}
}

func TestLogging_ErrorOnServerError(t *testing.T) {
	t.Parallel()
	log := &mockLogger{}
	handler := Logging(log)(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}),
	)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest("GET", "/fail", nil))
	found := findKV(log.args, "status")
	if found != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %v", found)
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
