package http

import (
	"strings"

	"github.com/shuldan/framework/pkg/contracts"
)

type httpRouterGroup struct {
	router     *httpRouter
	prefix     string
	middleware []contracts.HTTPMiddleware
}

func NewRouterGroup(router *httpRouter, prefix string, middleware []contracts.HTTPMiddleware) contracts.HTTPRouterGroup {
	return &httpRouterGroup{
		router:     router,
		prefix:     strings.TrimSuffix(prefix, "/"),
		middleware: middleware,
	}
}

func (g *httpRouterGroup) GET(path string, handler contracts.HTTPHandler, middleware ...contracts.HTTPMiddleware) {
	g.Handle("GET", path, handler, middleware...)
}

func (g *httpRouterGroup) POST(path string, handler contracts.HTTPHandler, middleware ...contracts.HTTPMiddleware) {
	g.Handle("POST", path, handler, middleware...)
}

func (g *httpRouterGroup) PUT(path string, handler contracts.HTTPHandler, middleware ...contracts.HTTPMiddleware) {
	g.Handle("PUT", path, handler, middleware...)
}

func (g *httpRouterGroup) DELETE(path string, handler contracts.HTTPHandler, middleware ...contracts.HTTPMiddleware) {
	g.Handle("DELETE", path, handler, middleware...)
}

func (g *httpRouterGroup) PATCH(path string, handler contracts.HTTPHandler, middleware ...contracts.HTTPMiddleware) {
	g.Handle("PATCH", path, handler, middleware...)
}

func (g *httpRouterGroup) HEAD(path string, handler contracts.HTTPHandler, middleware ...contracts.HTTPMiddleware) {
	g.Handle("HEAD", path, handler, middleware...)
}

func (g *httpRouterGroup) OPTIONS(path string, handler contracts.HTTPHandler, middleware ...contracts.HTTPMiddleware) {
	g.Handle("OPTIONS", path, handler, middleware...)
}

func (g *httpRouterGroup) Group(prefix string, middleware ...contracts.HTTPMiddleware) contracts.HTTPRouterGroup {
	newPrefix := g.prefix + "/" + strings.TrimPrefix(prefix, "/")
	newMiddleware := append(g.middleware, middleware...) //nolint:gocritic
	return NewRouterGroup(g.router, newPrefix, newMiddleware)
}

func (g *httpRouterGroup) Use(middleware ...contracts.HTTPMiddleware) {
	g.middleware = append(g.middleware, middleware...)
}

func (g *httpRouterGroup) Handle(method, path string, handler contracts.HTTPHandler, middleware ...contracts.HTTPMiddleware) {
	fullPath := g.prefix + "/" + strings.TrimPrefix(path, "/")
	fullMiddleware := append(g.middleware, middleware...) //nolint:gocritic
	g.router.Handle(method, fullPath, handler, fullMiddleware...)
}
