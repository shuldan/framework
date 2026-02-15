package httpserver

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPathParam(t *testing.T) {
	t.Parallel()
	router := NewRouter()
	router.GET("/items/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := PathParam(r, "id")
		_, _ = w.Write([]byte(id))
	})
	rr := serve(router, "GET", "/items/abc", nil)
	assertBody(t, "abc", rr)
}

func TestQueryParam(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest("GET", "/search?q=hello&page=2", nil)
	if v := QueryParam(r, "q"); v != "hello" {
		t.Errorf("expected 'hello', got %q", v)
	}
	if v := QueryParam(r, "page"); v != "2" {
		t.Errorf("expected '2', got %q", v)
	}
	if v := QueryParam(r, "missing"); v != "" {
		t.Errorf("expected empty, got %q", v)
	}
}

func TestBind_ValidJSON(t *testing.T) {
	t.Parallel()
	type input struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	body := strings.NewReader(`{"name":"Alice","age":30}`)
	r := httptest.NewRequest("POST", "/", body)
	var in input
	if err := Bind(r, &in); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if in.Name != "Alice" || in.Age != 30 {
		t.Fatalf("unexpected result: %+v", in)
	}
}

func TestBind_EmptyBody(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest("POST", "/", nil)
	var target map[string]any
	err := Bind(r, &target)
	if !errors.Is(err, ErrEmptyBody) {
		t.Fatalf("expected ErrEmptyBody, got %v", err)
	}
}

func TestBind_NoBody(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest("POST", "/", http.NoBody)
	var target map[string]any
	err := Bind(r, &target)
	if !errors.Is(err, ErrEmptyBody) {
		t.Fatalf("expected ErrEmptyBody, got %v", err)
	}
}

func TestBind_InvalidJSON(t *testing.T) {
	t.Parallel()
	body := strings.NewReader(`{invalid}`)
	r := httptest.NewRequest("POST", "/", body)
	var target map[string]any
	err := Bind(r, &target)
	if !errors.Is(err, ErrInvalidJSON) {
		t.Fatalf("expected ErrInvalidJSON, got %v", err)
	}
}

func TestBindWithLimit_TooLarge(t *testing.T) {
	t.Parallel()
	body := strings.NewReader(`{"data":"` + strings.Repeat("x", 100) + `"}`)
	r := httptest.NewRequest("POST", "/", body)
	var target map[string]any
	err := BindWithLimit(r, &target, 10)
	if !errors.Is(err, ErrBodyTooLarge) {
		t.Fatalf("expected ErrBodyTooLarge, got %v", err)
	}
}
