package middleware

import (
	"net/http"
	"time"
)

type LevelLogger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

func Logging(log LevelLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(sw, r)

			attrs := []any{
				"method", r.Method,
				"path", r.URL.Path,
				"status", sw.status,
				"duration", time.Since(start).String(),
				"request_id", IDFromContext(r.Context()),
			}

			logByStatus(log, sw.status, attrs)
		})
	}
}

func logByStatus(log LevelLogger, status int, attrs []any) {
	switch {
	case status >= http.StatusInternalServerError:
		log.Error("http request", attrs...)
	case status >= http.StatusBadRequest:
		log.Warn("http request", attrs...)
	default:
		log.Info("http request", attrs...)
	}
}

type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *statusWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.status = code
		w.wroteHeader = true
	}

	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
