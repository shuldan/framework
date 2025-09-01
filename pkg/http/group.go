package http

import (
	"strings"

	"github.com/shuldan/framework/pkg/contracts"
)

type RouterGroup struct {
	router     *Router
	prefix     string
	middleware []contracts.HTTPMiddleware
}

func NewRouterGroup(router *Router, prefix string, middleware []contracts.HTTPMiddleware) *RouterGroup {
	return &RouterGroup{
		router:     router,
		prefix:     strings.TrimSuffix(prefix, "/"),
		middleware: middleware,
	}
}

func (g *RouterGroup) GET(path string, handler contracts.HTTPHandler, middleware ...contracts.HTTPMiddleware) {
	g.Handle("GET", path, handler, middleware...)
}

func (g *RouterGroup) POST(path string, handler contracts.HTTPHandler, middleware ...contracts.HTTPMiddleware) {
	g.Handle("POST", path, handler, middleware...)
}

func (g *RouterGroup) PUT(path string, handler contracts.HTTPHandler, middleware ...contracts.HTTPMiddleware) {
	g.Handle("PUT", path, handler, middleware...)
}

func (g *RouterGroup) DELETE(path string, handler contracts.HTTPHandler, middleware ...contracts.HTTPMiddleware) {
	g.Handle("DELETE", path, handler, middleware...)
}

func (g *RouterGroup) PATCH(path string, handler contracts.HTTPHandler, middleware ...contracts.HTTPMiddleware) {
	g.Handle("PATCH", path, handler, middleware...)
}

func (g *RouterGroup) HEAD(path string, handler contracts.HTTPHandler, middleware ...contracts.HTTPMiddleware) {
	g.Handle("HEAD", path, handler, middleware...)
}

func (g *RouterGroup) OPTIONS(path string, handler contracts.HTTPHandler, middleware ...contracts.HTTPMiddleware) {
	g.Handle("OPTIONS", path, handler, middleware...)
}

func (g *RouterGroup) Group(prefix string, middleware ...contracts.HTTPMiddleware) contracts.HTTPRouterGroup {
	newPrefix := g.prefix + "/" + strings.TrimPrefix(prefix, "/")
	newMiddleware := append(g.middleware, middleware...) //nolint:gocritic
	return NewRouterGroup(g.router, newPrefix, newMiddleware)
}

func (g *RouterGroup) Use(middleware ...contracts.HTTPMiddleware) {
	g.middleware = append(g.middleware, middleware...)
}

func (g *RouterGroup) Handle(method, path string, handler contracts.HTTPHandler, middleware ...contracts.HTTPMiddleware) {
	fullPath := g.prefix + "/" + strings.TrimPrefix(path, "/")
	fullMiddleware := append(g.middleware, middleware...) //nolint:gocritic
	g.router.Handle(method, fullPath, handler, fullMiddleware...)
}
