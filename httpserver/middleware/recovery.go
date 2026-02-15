package middleware

import (
	"net/http"
	"runtime/debug"
)

const errBody = `{"code":"internal","message":"internal error"}`

func Recovery(
	log func(msg string, args ...any),
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer handlePanic(w, log)
			next.ServeHTTP(w, r)
		})
	}
}

func handlePanic(
	w http.ResponseWriter,
	log func(msg string, args ...any),
) {
	rec := recover()
	if rec == nil {
		return
	}

	if log != nil {
		log("panic recovered",
			"error", rec,
			"stack", string(debug.Stack()),
		)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = w.Write([]byte(errBody))
}
