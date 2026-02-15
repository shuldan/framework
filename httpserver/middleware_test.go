package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestApplyChain_Order(t *testing.T) {
	t.Parallel()
	var order []int
	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, 1)
			next.ServeHTTP(w, r)
		})
	}
	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, 2)
			next.ServeHTTP(w, r)
		})
	}
	handler := applyChain(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			order = append(order, 3)
			w.WriteHeader(http.StatusOK)
		}),
		[]Middleware{mw1, mw2},
	)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Errorf("expected [1 2 3], got %v", order)
	}
}

func TestApplyChain_Empty(t *testing.T) {
	t.Parallel()
	called := false
	handler := applyChain(
		http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { called = true }),
		nil,
	)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	if !called {
		t.Error("handler not called")
	}
}
