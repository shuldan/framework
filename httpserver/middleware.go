package httpserver

import "net/http"

type Middleware func(http.Handler) http.Handler

func applyChain(h http.Handler, mw []Middleware) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}

	return h
}
