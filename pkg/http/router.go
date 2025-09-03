package http

import (
	"context"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

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
	mu                      sync.RWMutex
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
	r.mu.Lock()
	defer r.mu.Unlock()
	r.middleware = append(r.middleware, middleware...)
}

func (r *Router) Static(path, root string) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		r.logger.Critical("Static: invalid root directory", "root", root, "error", err)
		return
	}
	if info, err := os.Stat(absRoot); err != nil || !info.IsDir() {
		r.logger.Critical("Static: root directory doesn't exist or is not a directory", "root", absRoot)
		return
	}
	r.GET(path+"/*", func(ctx contracts.HTTPContext) error {
		filePath := ctx.Param("*")
		if strings.ContainsAny(filePath, "\x00") {
			return ErrFileNotFound.WithDetail("path", filePath).WithDetail("reason", "invalid characters")
		}
		if !isPathSafe(absRoot, filePath) {
			return ErrFileNotFound.WithDetail("path", filePath).WithDetail("reason", "unsafe path")
		}
		fullPath := filepath.Join(absRoot, filepath.Clean(filePath))
		if rel, err := filepath.Rel(absRoot, fullPath); err != nil || strings.HasPrefix(rel, "../") {
			return ErrFileNotFound.WithDetail("path", filePath).WithDetail("reason", "path outside root")
		}
		info, err := os.Stat(fullPath)
		if err != nil {
			return ErrFileNotFound.WithDetail("path", fullPath).WithCause(err)
		}
		if info.IsDir() {
			indexPath := filepath.Join(fullPath, "index.html")
			if !isPathSafe(absRoot, filepath.Join(filePath, "index.html")) {
				return ErrFileNotFound.WithDetail("path", indexPath).WithDetail("reason", "unsafe index path")
			}
			if _, err := os.Stat(indexPath); err != nil {
				return ErrFileNotFound.WithDetail("path", indexPath)
			}
			fullPath = indexPath
		}
		const maxFileSize = 100 << 20
		if info.Size() > maxFileSize {
			return ErrFileNotFound.WithDetail("path", fullPath).WithDetail("reason", "file too large")
		}
		data, err := os.ReadFile(filepath.Clean(fullPath))
		if err != nil {
			return ErrFileNotFound.WithDetail("path", fullPath).WithCause(err)
		}
		contentType := mime.TypeByExtension(filepath.Ext(fullPath))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		ctx.SetHeader("X-Content-Type-Options", "nosniff")
		ctx.SetHeader("X-Frame-Options", "DENY")
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

	r.mu.Lock()
	defer r.mu.Unlock()

	r.routes = append(r.routes, Route{
		method:     method,
		pattern:    path,
		handler:    handler,
		middleware: middleware,
	})
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := NewHTTPContext(w, req, r.logger)
	req = req.WithContext(context.WithValue(req.Context(), ContextKey, ctx))

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
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, route := range r.routes {
		if route.method != method {
			continue
		}
		if params := r.matchPattern(route.pattern, path); params != nil {
			routeCopy := route
			return &routeCopy, params
		}
	}
	return nil, nil
}

func (r *Router) pathExistsWithDifferentMethod(path, method string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

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
	if strings.HasPrefix(cleanPath, "../") || cleanPath == ".." || strings.Contains(cleanPath, "/../") {
		return false
	}
	if root == "" {
		return !strings.Contains(cleanPath, "..")
	}
	fullPath := filepath.Join(root, cleanPath)
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absFullPath)
	if err != nil {
		return false
	}
	if strings.HasPrefix(rel, "../") || rel == ".." {
		return false
	}
	if info, err := os.Lstat(absFullPath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := filepath.EvalSymlinks(absFullPath)
			if err != nil {
				return false
			}
			rel, err := filepath.Rel(absRoot, target)
			if err != nil {
				return false
			}
			if strings.HasPrefix(rel, "../") || rel == ".." {
				return false
			}
		}
	}
	return true
}
