package httpserver

import "net/http"

type Router struct {
	mux        *http.ServeMux
	prefix     string
	middleware []Middleware
}

func NewRouter() *Router {
	return &Router{mux: http.NewServeMux()}
}

func (rt *Router) Use(mw ...Middleware) {
	rt.middleware = append(rt.middleware, mw...)
}

func (rt *Router) Group(
	prefix string, mw ...Middleware,
) *Router {
	combined := make(
		[]Middleware, len(rt.middleware), len(rt.middleware)+len(mw),
	)
	copy(combined, rt.middleware)
	combined = append(combined, mw...)

	return &Router{
		mux:        rt.mux,
		prefix:     rt.prefix + prefix,
		middleware: combined,
	}
}

func (rt *Router) GET(pattern string, h http.HandlerFunc) {
	rt.handle("GET", pattern, h)
}

func (rt *Router) POST(pattern string, h http.HandlerFunc) {
	rt.handle("POST", pattern, h)
}

func (rt *Router) PUT(pattern string, h http.HandlerFunc) {
	rt.handle("PUT", pattern, h)
}

func (rt *Router) PATCH(pattern string, h http.HandlerFunc) {
	rt.handle("PATCH", pattern, h)
}

func (rt *Router) DELETE(pattern string, h http.HandlerFunc) {
	rt.handle("DELETE", pattern, h)
}

func (rt *Router) Handle(
	method, pattern string, h http.Handler,
) {
	rt.handle(method, pattern, h)
}

func (rt *Router) ServeHTTP(
	w http.ResponseWriter, r *http.Request,
) {
	rt.mux.ServeHTTP(w, r)
}

func (rt *Router) handle(
	method, pattern string, h http.Handler,
) {
	full := method + " " + rt.prefix + pattern
	rt.mux.Handle(full, applyChain(h, rt.middleware))
}
