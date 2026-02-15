package httpserver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouter_GET(t *testing.T) {
	t.Parallel()
	router := NewRouter()
	router.GET("/ping", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("pong"))
	})
	rr := serve(router, "GET", "/ping", nil)
	assertBody(t, "pong", rr)
	assertStatus(t, http.StatusOK, rr)
}

func TestRouter_Methods(t *testing.T) {
	t.Parallel()
	methods := []struct {
		register func(*Router, string, http.HandlerFunc)
		method   string
	}{
		{(*Router).GET, "GET"},
		{(*Router).POST, "POST"},
		{(*Router).PUT, "PUT"},
		{(*Router).PATCH, "PATCH"},
		{(*Router).DELETE, "DELETE"},
	}
	for _, m := range methods {
		t.Run(m.method, func(t *testing.T) {
			t.Parallel()
			router := NewRouter()
			m.register(router, "/test", ok)
			rr := serve(router, m.method, "/test", nil)
			assertStatus(t, http.StatusOK, rr)
		})
	}
}

func TestRouter_Handle(t *testing.T) {
	t.Parallel()
	router := NewRouter()
	router.Handle("GET", "/custom", http.HandlerFunc(ok))
	rr := serve(router, "GET", "/custom", nil)
	assertStatus(t, http.StatusOK, rr)
}

func TestRouter_Middleware(t *testing.T) {
	t.Parallel()
	router := NewRouter()
	router.Use(headerMiddleware("X-Test", "applied"))
	router.GET("/test", ok)
	rr := serve(router, "GET", "/test", nil)
	assertHeader(t, "X-Test", "applied", rr)
}

func TestRouter_Group_Prefix(t *testing.T) {
	t.Parallel()
	router := NewRouter()
	api := router.Group("/api")
	api.GET("/users", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("users"))
	})
	rr := serve(router, "GET", "/api/users", nil)
	assertBody(t, "users", rr)
}

func TestRouter_Group_Middleware(t *testing.T) {
	t.Parallel()
	router := NewRouter()
	router.Use(headerMiddleware("X-Global", "yes"))
	api := router.Group("/api", headerMiddleware("X-Group", "yes"))
	api.GET("/test", ok)
	rr := serve(router, "GET", "/api/test", nil)
	assertHeader(t, "X-Global", "yes", rr)
	assertHeader(t, "X-Group", "yes", rr)
}

func TestRouter_PathParam(t *testing.T) {
	t.Parallel()
	router := NewRouter()
	router.GET("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.PathValue("id")))
	})
	rr := serve(router, "GET", "/users/42", nil)
	assertBody(t, "42", rr)
}

func ok(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func serve(h http.Handler, method, path string, body io.Reader) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, body)
	h.ServeHTTP(rr, req)
	return rr
}

func headerMiddleware(key, val string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(key, val)
			next.ServeHTTP(w, r)
		})
	}
}

func assertBody(t *testing.T, expected string, rr *httptest.ResponseRecorder) {
	t.Helper()
	if rr.Body.String() != expected {
		t.Errorf("body: expected %q, got %q", expected, rr.Body.String())
	}
}

func assertStatus(t *testing.T, expected int, rr *httptest.ResponseRecorder) {
	t.Helper()
	if rr.Code != expected {
		t.Errorf("status: expected %d, got %d", expected, rr.Code)
	}
}

func assertHeader(t *testing.T, key, expected string, rr *httptest.ResponseRecorder) {
	t.Helper()
	if v := rr.Header().Get(key); v != expected {
		t.Errorf("header %s: expected %q, got %q", key, expected, v)
	}
}
