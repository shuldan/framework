package middleware

import (
	"net/http"
	"strconv"
	"strings"
)

type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
	MaxAge         int // seconds
}

func CORS(cfg CORSConfig) func(http.Handler) http.Handler {
	methods := strings.Join(cfg.AllowedMethods, ", ")
	headers := strings.Join(cfg.AllowedHeaders, ", ")
	maxAge := strconv.Itoa(cfg.MaxAge)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			matched, allowOrigin := matchOrigin(origin, cfg.AllowedOrigins)
			if !matched {
				next.ServeHTTP(w, r)
				return
			}

			setCORSHeaders(w, allowOrigin, methods, headers, maxAge)

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func matchOrigin(
	origin string, allowed []string,
) (bool, string) {
	for _, a := range allowed {
		if a == "*" {
			return true, "*"
		}

		if a == origin {
			return true, origin
		}
	}

	return false, ""
}

func setCORSHeaders(
	w http.ResponseWriter,
	origin, methods, headers, maxAge string,
) {
	w.Header().Set("Access-Control-Allow-Origin", origin)

	if methods != "" {
		w.Header().Set("Access-Control-Allow-Methods", methods)
	}

	if headers != "" {
		w.Header().Set("Access-Control-Allow-Headers", headers)
	}

	if maxAge != "0" {
		w.Header().Set("Access-Control-Max-Age", maxAge)
	}
}
