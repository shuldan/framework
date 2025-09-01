package http

import (
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/shuldan/framework/pkg/contracts"
)

type Route struct {
	method     string
	pattern    string
	handler    contracts.HTTPHandler
	middleware []contracts.HTTPMiddleware
}

type Router struct {
	routes                  []Route
	middleware              []contracts.HTTPMiddleware
	errorHandler            contracts.HTTPErrorHandler
	logger                  contracts.Logger
	notFoundHandler         contracts.HTTPHandler
	methodNotAllowedHandler contracts.HTTPHandler
}

func NewRouter(logger contracts.Logger) *Router {
	r := &Router{
		routes: make([]Route, 0),
		logger: logger,
	}

	r.notFoundHandler = func(ctx contracts.HTTPContext) error {
		return ErrRouteNotFound.
			WithDetail("method", ctx.Method()).
			WithDetail("path", ctx.Path())
	}

	r.methodNotAllowedHandler = func(ctx contracts.HTTPContext) error {
		return ErrMethodNotAllowed.
			WithDetail("method", ctx.Method()).
			WithDetail("path", ctx.Path())
	}

	r.errorHandler = func(ctx contracts.HTTPContext, err error) {
		if ctx.StatusCode() == 0 {
			ctx.Status(http.StatusInternalServerError)
		}

		if r.logger != nil {
			r.logger.Error("HTTP handler error", "error", err, "path", ctx.Path(), "method", ctx.Method())
		}

		if err = ctx.JSON(map[string]string{"error": err.Error()}); err != nil && r.logger != nil {
			r.logger.Error("JSON error", "error", err, "path", ctx.Path(), "method", ctx.Method())
		}
	}

	return r
}

func (r *Router) GET(path string, handler contracts.HTTPHandler, middleware ...contracts.HTTPMiddleware) {
	r.Handle("GET", path, handler, middleware...)
}

func (r *Router) POST(path string, handler contracts.HTTPHandler, middleware ...contracts.HTTPMiddleware) {
	r.Handle("POST", path, handler, middleware...)
}

func (r *Router) PUT(path string, handler contracts.HTTPHandler, middleware ...contracts.HTTPMiddleware) {
	r.Handle("PUT", path, handler, middleware...)
}

func (r *Router) DELETE(path string, handler contracts.HTTPHandler, middleware ...contracts.HTTPMiddleware) {
	r.Handle("DELETE", path, handler, middleware...)
}

func (r *Router) PATCH(path string, handler contracts.HTTPHandler, middleware ...contracts.HTTPMiddleware) {
	r.Handle("PATCH", path, handler, middleware...)
}

func (r *Router) HEAD(path string, handler contracts.HTTPHandler, middleware ...contracts.HTTPMiddleware) {
	r.Handle("HEAD", path, handler, middleware...)
}

func (r *Router) OPTIONS(path string, handler contracts.HTTPHandler, middleware ...contracts.HTTPMiddleware) {
	r.Handle("OPTIONS", path, handler, middleware...)
}

func (r *Router) Group(prefix string, middleware ...contracts.HTTPMiddleware) contracts.HTTPRouterGroup {
	return NewRouterGroup(r, prefix, middleware)
}

func (r *Router) Use(middleware ...contracts.HTTPMiddleware) {
	r.middleware = append(r.middleware, middleware...)
}

func (r *Router) Static(path, root string) {
	r.GET(path+"/*", func(ctx contracts.HTTPContext) error {
		filePath := ctx.Param("*")
		if !isPathSafe(root, filePath) {
			return ErrFileNotFound.WithDetail("path", filePath)
		}

		fullPath := filepath.Join(root, filepath.Clean(filePath))

		info, err := os.Stat(fullPath)
		if err != nil {
			return ErrFileNotFound.WithDetail("path", fullPath).WithCause(err)
		}

		if info.IsDir() {
			fullPath = filepath.Join(fullPath, "index.html")
			if _, err := os.Stat(fullPath); err != nil {
				return ErrFileNotFound.WithDetail("path", fullPath)
			}
		}

		data, err := os.ReadFile(filepath.Clean(fullPath))
		if err != nil {
			return ErrFileNotFound.WithDetail("path", fullPath).WithCause(err)
		}

		contentType := mime.TypeByExtension(filepath.Ext(fullPath))
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		return ctx.Data(contentType, data)
	})
}

func (r *Router) StaticFile(path, fPath string) {
	if !isPathSafe("", fPath) {
		r.logger.Critical("StaticFile: unsafe file path " + fPath)
		return
	}

	r.GET(path, func(ctx contracts.HTTPContext) error {
		data, err := os.ReadFile(filepath.Clean(fPath))
		if err != nil {
			return ErrFileNotFound.WithDetail("path", fPath).WithCause(err)
		}

		contentType := mime.TypeByExtension(filepath.Ext(fPath))
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		return ctx.Data(contentType, data)
	})
}

func (r *Router) Handle(method, path string, handler contracts.HTTPHandler, middleware ...contracts.HTTPMiddleware) {
	if handler == nil {
		panic(ErrInvalidHandler)
	}

	r.routes = append(r.routes, Route{
		method:     method,
		pattern:    path,
		handler:    handler,
		middleware: middleware,
	})
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := NewHTTPContext(w, req, r.logger)

	route, params := r.matchRoute(req.Method, req.URL.Path)
	if route == nil {
		if r.pathExistsWithDifferentMethod(req.URL.Path, req.Method) {
			r.errorHandler(ctx, r.methodNotAllowedHandler(ctx))
		} else {
			r.errorHandler(ctx, r.notFoundHandler(ctx))
		}
		return
	}

	for key, value := range params {
		ctx.Set(key, value)
	}

	handler := route.handler

	for i := len(route.middleware) - 1; i >= 0; i-- {
		handler = route.middleware[i](handler)
	}

	for i := len(r.middleware) - 1; i >= 0; i-- {
		handler = r.middleware[i](handler)
	}

	if err := handler(ctx); err != nil {
		r.errorHandler(ctx, err)
	}
}

func (r *Router) matchRoute(method, path string) (*Route, map[string]string) {
	for _, route := range r.routes {
		if route.method != method {
			continue
		}

		if params := r.matchPattern(route.pattern, path); params != nil {
			return &route, params
		}
	}
	return nil, nil
}

func (r *Router) pathExistsWithDifferentMethod(path, method string) bool {
	for _, route := range r.routes {
		if route.method == method {
			continue
		}

		if r.matchPattern(route.pattern, path) != nil {
			return true
		}
	}
	return false
}

func (r *Router) matchPattern(pattern, path string) map[string]string {
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")

	params := make(map[string]string)

	if len(patternParts) > 0 && patternParts[len(patternParts)-1] == "*" {
		if len(pathParts) < len(patternParts)-1 {
			return nil
		}

		for i := 0; i < len(patternParts)-1; i++ {
			part := patternParts[i]
			if strings.HasPrefix(part, ":") {
				params[part[1:]] = pathParts[i]
			} else if part != pathParts[i] {
				return nil
			}
		}

		wildcardPath := strings.Join(pathParts[len(patternParts)-1:], "/")
		params["*"] = wildcardPath

		return params
	}

	if len(patternParts) != len(pathParts) {
		return nil
	}

	for i, part := range patternParts {
		if strings.HasPrefix(part, ":") {
			params[part[1:]] = pathParts[i]
		} else if part != pathParts[i] {
			return nil
		}
	}

	return params
}

func isPathSafe(root, path string) bool {
	if filepath.IsAbs(path) {
		return false
	}
	cleanPath := filepath.Clean(path)
	if strings.HasPrefix(cleanPath, "../") || cleanPath == ".." {
		return false
	}
	fullPath := filepath.Join(root, cleanPath)
	rel, err := filepath.Rel(root, fullPath)
	if err != nil {
		return false
	}
	if strings.HasPrefix(rel, "../") || rel == ".." {
		return false
	}

	return true
}
