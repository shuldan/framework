package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shuldan/framework/pkg/contracts"
)

func TestRouterGroupAllMethods(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	router := NewRouter(logger).(*httpRouter)
	group := NewRouterGroup(router, "/api", nil)

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	handlerCalls := make(map[string]bool)

	for _, method := range methods {
		method := method
		handler := func(ctx contracts.HTTPContext) error {
			handlerCalls[method] = true
			return ctx.JSON(map[string]string{"method": method})
		}

		switch method {
		case http.MethodGet:
			group.GET("/test", handler)
		case http.MethodPost:
			group.POST("/test", handler)
		case http.MethodPut:
			group.PUT("/test", handler)
		case http.MethodDelete:
			group.DELETE("/test", handler)
		case http.MethodPatch:
			group.PATCH("/test", handler)
		case http.MethodHead:
			group.HEAD("/test", handler)
		case http.MethodOptions:
			group.OPTIONS("/test", handler)
		}

		req := httptest.NewRequest(method, "/api/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if !handlerCalls[method] {
			t.Errorf("%s handler was not called", method)
		}
	}
}

func TestRouterGroupUse(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	router := NewRouter(logger).(*httpRouter)
	group := NewRouterGroup(router, "/api", nil)

	middlewareCalled := false
	middleware := func(next contracts.HTTPHandler) contracts.HTTPHandler {
		return func(ctx contracts.HTTPContext) error {
			middlewareCalled = true
			return next(ctx)
		}
	}

	group.Use(middleware)

	handlerCalled := false
	handler := func(ctx contracts.HTTPContext) error {
		handlerCalled = true
		return ctx.JSON(map[string]string{"status": "ok"})
	}

	group.GET("/test", handler)

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !middlewareCalled {
		t.Error("Group middleware was not called")
	}
	if !handlerCalled {
		t.Error("Handler was not called")
	}
}

func TestRouterGroupNestedWithSlashes(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	router := NewRouter(logger).(*httpRouter)

	api := router.Group("/api/")
	v1 := api.Group("/v1/")
	users := v1.Group("/users/")

	handlerCalled := false
	handler := func(ctx contracts.HTTPContext) error {
		handlerCalled = true
		return ctx.JSON(map[string]string{"status": "ok"})
	}

	users.GET("/profile", handler)

	req := httptest.NewRequest("GET", "/api/v1/users/profile", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("Nested group handler was not called")
	}
}

func TestRouterGroupHandle(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	router := NewRouter(logger).(*httpRouter)
	group := NewRouterGroup(router, "/api", nil)

	handlerCalled := false
	handler := func(ctx contracts.HTTPContext) error {
		handlerCalled = true
		return ctx.JSON(map[string]string{"status": "ok"})
	}

	group.Handle("CUSTOM", "/test", handler)

	req := httptest.NewRequest("CUSTOM", "/api/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("Custom method handler was not called")
	}
}
