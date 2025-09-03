package http

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/shuldan/framework/pkg/contracts"
)

func LoggingMiddleware(logger contracts.Logger) contracts.HTTPMiddleware {
	return func(next contracts.HTTPHandler) contracts.HTTPHandler {
		return func(ctx contracts.HTTPContext) error {
			start := time.Now()
			logCtx := context.WithValue(ctx.Context(), RequestStart, start)
			ctx.SetContext(logCtx)
			err := next(ctx)
			duration := time.Since(start)
			status := ctx.StatusCode()
			if status == 0 {
				status = 200
			}
			logLevel := "info"
			if status >= 400 && status < 500 {
				logLevel = "warn"
			} else if status >= 500 {
				logLevel = "error"
			}
			logArgs := []any{
				"method", ctx.Method(),
				"path", ctx.Path(),
				"status", status,
				"duration", duration,
				"request_id", ctx.RequestID(),
			}
			if userID, exists := ctx.Get("user_id"); exists {
				logArgs = append(logArgs, "user_id", userID)
			}
			if userAgent := ctx.RequestHeader("User-Agent"); userAgent != "" {
				logArgs = append(logArgs, "user_agent", userAgent)
			}
			if forwarded := ctx.RequestHeader("X-Forwarded-For"); forwarded != "" {
				logArgs = append(logArgs, "client_ip", strings.Split(forwarded, ",")[0])
			}
			switch logLevel {
			case "warn":
				logger.Warn("HTTP request completed with client error", logArgs...)
			case "error":
				logger.Error("HTTP request completed with server error", logArgs...)
			default:
				logger.Info("HTTP request completed", logArgs...)
			}
			if err != nil {
				logger.Error("HTTP request error",
					append(logArgs, "error", err.Error())...,
				)
			}
			return err
		}
	}
}

func RecoveryMiddleware(logger contracts.Logger) contracts.HTTPMiddleware {
	return func(next contracts.HTTPHandler) contracts.HTTPHandler {
		return func(ctx contracts.HTTPContext) (err error) {
			defer func() {
				if r := recover(); r != nil {
					stack := make([]byte, 4096)
					length := runtime.Stack(stack, false)
					stackTrace := string(stack[:length])
					if logger != nil {
						logger.Error("HTTP handler panic",
							"panic", r,
							"method", ctx.Method(),
							"path", ctx.Path(),
							"request_id", ctx.RequestID(),
							"stack_trace", stackTrace,
						)
					}
					if ctx.StatusCode() == 0 {
						response := map[string]interface{}{
							"error":      "Internal server error",
							"request_id": ctx.RequestID(),
							"timestamp":  time.Now().UTC().Format(time.RFC3339),
						}
						if jsonErr := ctx.Status(http.StatusInternalServerError).JSON(response); jsonErr != nil && logger != nil {
							logger.Error("Failed to send panic response", "error", jsonErr)
						}
					}
					err = fmt.Errorf("panic recovered: %v", r)
				}
			}()
			return next(ctx)
		}
	}
}

func RequestIDMiddleware() contracts.HTTPMiddleware {
	return func(next contracts.HTTPHandler) contracts.HTTPHandler {
		return func(ctx contracts.HTTPContext) error {
			ctx.SetHeader("X-Request-ID", ctx.RequestID())
			return next(ctx)
		}
	}
}

func ErrorHandlerMiddleware(errorHandler contracts.ErrorHandler) contracts.HTTPMiddleware {
	return func(next contracts.HTTPHandler) contracts.HTTPHandler {
		return func(ctx contracts.HTTPContext) error {
			if err := next(ctx); err != nil {
				return errorHandler.Handle(ctx.Context(), err)
			}
			return nil
		}
	}
}

type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	AllowCredentials bool
	MaxAge           time.Duration
}

func CORSMiddleware(config CORSConfig) contracts.HTTPMiddleware {
	if len(config.AllowMethods) == 0 {
		config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD", "PATCH"}
	}
	if len(config.AllowHeaders) == 0 {
		config.AllowHeaders = []string{
			"Origin", "Content-Length", "Content-Type", "Authorization",
			"X-Requested-With", "Accept", "Accept-Language", "Accept-Encoding",
		}
	}
	if config.MaxAge == 0 {
		config.MaxAge = 24 * time.Hour
	}
	return func(next contracts.HTTPHandler) contracts.HTTPHandler {
		return func(ctx contracts.HTTPContext) error {
			origin := ctx.RequestHeader("Origin")
			if origin == "" {
				return next(ctx)
			}
			allowedOrigin, valid := checkOrigin(config, origin)
			if !valid {
				ctx.Status(http.StatusForbidden)
				return ctx.JSON(map[string]interface{}{
					"error":      "CORS origin not allowed",
					"origin":     origin,
					"request_id": ctx.RequestID(),
					"timestamp":  time.Now().UTC().Format(time.RFC3339),
				})
			}
			ctx.SetHeader("Access-Control-Allow-Origin", allowedOrigin)
			ctx.SetHeader("Vary", "Origin")
			if config.AllowCredentials {
				ctx.SetHeader("Access-Control-Allow-Credentials", "true")
			}
			if ctx.Method() == "OPTIONS" {
				ctx.SetHeader("Access-Control-Allow-Methods", strings.Join(config.AllowMethods, ", "))
				ctx.SetHeader("Access-Control-Allow-Headers", strings.Join(config.AllowHeaders, ", "))
				ctx.SetHeader("Access-Control-Max-Age", fmt.Sprintf("%d", int64(config.MaxAge.Seconds())))
				return ctx.Status(http.StatusNoContent).NoContent()
			}
			ctx.SetHeader("Access-Control-Expose-Headers", "Content-Length, X-Request-ID")
			return next(ctx)
		}
	}
}

func checkOrigin(config CORSConfig, origin string) (string, bool) {
	if origin == "" {
		return "", false
	}
	if _, err := url.Parse(origin); err != nil {
		return "", false
	}
	switch {
	case len(config.AllowOrigins) == 0:
		return origin, true
	case len(config.AllowOrigins) == 1 && config.AllowOrigins[0] == "*":
		if config.AllowCredentials {
			return origin, true
		}
		return "*", true
	default:
		for _, allowedOrigin := range config.AllowOrigins {
			if allowedOrigin == origin {
				return origin, true
			}
			if strings.HasPrefix(allowedOrigin, "*.") {
				domain := allowedOrigin[2:]
				if strings.HasSuffix(origin, "."+domain) || origin == "https://"+domain || origin == "http://"+domain {
					return origin, true
				}
			}
		}
		return "", false
	}
}

func SecurityMiddleware() contracts.HTTPMiddleware {
	return func(next contracts.HTTPHandler) contracts.HTTPHandler {
		return func(ctx contracts.HTTPContext) error {
			ctx.SetHeader("X-Content-Type-Options", "nosniff")
			ctx.SetHeader("X-Frame-Options", "DENY")
			ctx.SetHeader("X-XSS-Protection", "1; mode=block")
			ctx.SetHeader("Referrer-Policy", "strict-origin-when-cross-origin")
			ctx.SetHeader("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'")
			ctx.SetHeader("Server", "")
			return next(ctx)
		}
	}
}

func HSTSMiddleware(maxAge time.Duration, includeSubdomains bool) contracts.HTTPMiddleware {
	return func(next contracts.HTTPHandler) contracts.HTTPHandler {
		return func(ctx contracts.HTTPContext) error {
			if ctx.Request().TLS != nil {
				hstsValue := fmt.Sprintf("max-age=%d", int64(maxAge.Seconds()))
				if includeSubdomains {
					hstsValue += "; includeSubDomains"
				}
				hstsValue += "; preload"

				ctx.SetHeader("Strict-Transport-Security", hstsValue)
			}
			return next(ctx)
		}
	}
}
