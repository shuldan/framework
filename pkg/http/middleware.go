package http

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/shuldan/framework/pkg/contracts"
	"github.com/shuldan/framework/pkg/errors"
)

func LoggingMiddleware(logger contracts.Logger) contracts.HTTPMiddleware {
	return func(next contracts.HTTPHandler) contracts.HTTPHandler {
		return func(ctx contracts.HTTPContext) error {
			start := time.Now()

			err := next(ctx)

			duration := time.Since(start)
			status := ctx.StatusCode()
			if status == 0 {
				status = 200
			}

			logger.Info("HTTP request",
				"method", ctx.Method(),
				"path", ctx.Path(),
				"status", status,
				"duration", duration,
				"request_id", ctx.RequestID(),
			)

			if err != nil {
				logger.Error("HTTP request error",
					"error", err,
					"method", ctx.Method(),
					"path", ctx.Path(),
					"request_id", ctx.RequestID(),
				)
			}

			return err
		}
	}
}

func RecoveryMiddleware(logger contracts.Logger) contracts.HTTPMiddleware {
	return func(next contracts.HTTPHandler) contracts.HTTPHandler {
		return func(ctx contracts.HTTPContext) error {
			defer func() {
				if r := recover(); r != nil {
					if logger != nil {
						logger.Error("HTTP handler panic",
							"panic", r,
							"method", ctx.Method(),
							"path", ctx.Path(),
							"request_id", ctx.RequestID(),
						)
					}

					if ctx.StatusCode() == 0 {
						err := ctx.Status(http.StatusInternalServerError).JSON(map[string]string{
							"error": "Internal server error",
						})
						if err != nil && logger != nil {
							logger.Error("HTTP request error", "error", err)
						}
					}
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
			requestCtx := context.WithValue(ctx.Context(), errors.HTTPContextKey, ctx)
			ctx.SetContext(requestCtx)
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
	return func(next contracts.HTTPHandler) contracts.HTTPHandler {
		return func(ctx contracts.HTTPContext) error {
			origin := ctx.RequestHeader("Origin")
			if origin == "" {
				return next(ctx)
			}

			allowedOrigin, valid := checkOrigin(config, origin)
			if !valid {
				return ctx.Status(http.StatusForbidden).JSON(map[string]string{
					"error": "CORS origin not allowed",
				})
			}

			ctx.SetHeader("Access-Control-Allow-Origin", allowedOrigin)

			if config.AllowCredentials {
				ctx.SetHeader("Access-Control-Allow-Credentials", "true")
			}

			if len(config.AllowMethods) > 0 {
				ctx.SetHeader("Access-Control-Allow-Methods", strings.Join(config.AllowMethods, ", "))
			}

			if len(config.AllowHeaders) > 0 {
				ctx.SetHeader("Access-Control-Allow-Headers", strings.Join(config.AllowHeaders, ", "))
			}

			if config.MaxAge > 0 {
				ctx.SetHeader("Access-Control-Max-Age", fmt.Sprintf("%d", int64(config.MaxAge.Seconds())))
			}

			if ctx.Method() == "OPTIONS" {
				return ctx.Status(http.StatusNoContent).NoContent()
			}

			return next(ctx)
		}
	}
}

func checkOrigin(config CORSConfig, origin string) (string, bool) {
	switch {
	case len(config.AllowOrigins) == 0:
		return "*", true

	case config.AllowOrigins[0] == "*":
		if config.AllowCredentials {
			return origin, true
		}
		return "*", true

	default:
		for _, o := range config.AllowOrigins {
			if o == origin {
				return origin, true
			}
		}
		return "", false
	}
}
