package http

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/shuldan/framework/pkg/contracts"
)

func LoadMiddlewareFromConfig(config contracts.Config, logger contracts.Logger) []contracts.HTTPMiddleware {
	var middlewares []contracts.HTTPMiddleware

	if sub, ok := config.GetSub("http.server.middleware"); ok {
		if m := loadSecurityHeadersMiddleware(sub, logger); m != nil {
			middlewares = append(middlewares, m)
		}
		if m := loadHSTSMiddleware(sub, logger); m != nil {
			middlewares = append(middlewares, m)
		}
		if m := loadCORSMiddleware(sub, logger); m != nil {
			middlewares = append(middlewares, m)
		}
		if m := loadLoggingMiddleware(sub, logger); m != nil {
			middlewares = append(middlewares, m)
		}
		if m := loadErrorHandlerMiddleware(sub, logger); m != nil {
			middlewares = append(middlewares, m)
		}
	}

	return middlewares
}

func loadSecurityHeadersMiddleware(sub contracts.Config, logger contracts.Logger) contracts.HTTPMiddleware {
	if secSub, ok := sub.GetSub("security_headers"); ok && secSub.GetBool("enabled", true) {
		logger.Info("Security headers middleware enabled")
		cfg := securityHeadersConfig{
			enabled:        true,
			csp:            secSub.GetString("csp", "default-src 'self';"),
			xFrameOptions:  secSub.GetString("x_frame_options", "DENY"),
			xXSSProtection: secSub.GetString("x_xss_protection", "1; mode=block"),
			referrerPolicy: secSub.GetString("referrer_policy", "strict-origin-when-cross-origin"),
		}
		return SecurityHeadersMiddleware(cfg)
	}
	return nil
}

func loadHSTSMiddleware(sub contracts.Config, logger contracts.Logger) contracts.HTTPMiddleware {
	if hstsSub, ok := sub.GetSub("hsts"); ok && hstsSub.GetBool("enabled", false) {
		logger.Info("HSTS middleware enabled")
		maxAge := time.Duration(hstsSub.GetInt("max_age", 31536000)) * time.Second
		cfg := hstsConfig{
			enabled:           true,
			maxAge:            maxAge,
			includeSubdomains: hstsSub.GetBool("include_subdomains", false),
			preload:           hstsSub.GetBool("preload", false),
		}
		return HSTSMiddleware(cfg)
	}
	return nil
}

func loadCORSMiddleware(sub contracts.Config, logger contracts.Logger) contracts.HTTPMiddleware {
	if corsSub, ok := sub.GetSub("cors"); ok && corsSub.GetBool("enabled", false) {
		logger.Info("CORS middleware enabled")
		cfg := CORSConfig{
			AllowOrigins:     corsSub.GetStringSlice("allow_origins"),
			AllowMethods:     corsSub.GetStringSlice("allow_methods"),
			AllowHeaders:     corsSub.GetStringSlice("allow_headers"),
			AllowCredentials: corsSub.GetBool("allow_credentials", false),
			MaxAge:           time.Duration(corsSub.GetInt("max_age", 86400)) * time.Second,
		}
		if len(cfg.AllowMethods) == 0 {
			cfg.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD", "PATCH"}
		}
		if len(cfg.AllowHeaders) == 0 {
			cfg.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-Requested-With"}
		}
		return CORSMiddleware(cfg)
	}
	return nil
}

func loadLoggingMiddleware(sub contracts.Config, logger contracts.Logger) contracts.HTTPMiddleware {
	if logSub, ok := sub.GetSub("logging"); ok && logSub.GetBool("enabled", false) {
		logger.Info("Logging middleware enabled")
		return LoggingMiddleware(logger)
	}
	return nil
}

func loadErrorHandlerMiddleware(sub contracts.Config, logger contracts.Logger) contracts.HTTPMiddleware {
	if errSub, ok := sub.GetSub("error_handler"); ok && errSub.GetBool("enabled", false) {
		logger.Info("Error handler middleware enabled")
		cfg := NewErrorHandlerConfig().
			WithShowStackTrace(errSub.GetBool("show_stack_trace", false)).
			WithShowDetails(errSub.GetBool("show_details", false)).
			WithLogLevel(errSub.GetString("log_level", "error"))

		if statusSub, ok := errSub.GetSub("status_codes"); ok {
			statusMap := make(map[string]int)
			for code := range statusSub.All() {
				statusMap[code] = statusSub.GetInt(code)
			}
			cfg = cfg.WithStatusCodes(statusMap)
		}

		if msgSub, ok := errSub.GetSub("user_messages"); ok {
			msgMap := make(map[string]string)
			for code := range msgSub.All() {
				msgMap[code] = msgSub.GetString(code)
			}
			cfg = cfg.WithUserMessages(msgMap)
		}

		return ErrorHandlerMiddleware(NewErrorHandler(cfg, logger))
	}
	return nil
}

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
				_ = errorHandler.Handle(context.WithValue(ctx.Context(), ContextKey, ctx), err)
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
	u, err := url.Parse(origin)
	if err != nil || u.Scheme == "" || u.Host == "" {
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

type securityHeadersConfig struct {
	enabled        bool
	csp            string
	xFrameOptions  string
	xXSSProtection string
	referrerPolicy string
}

func SecurityHeadersMiddleware(config securityHeadersConfig) contracts.HTTPMiddleware {
	return func(next contracts.HTTPHandler) contracts.HTTPHandler {
		return func(ctx contracts.HTTPContext) error {
			ctx.SetHeader("X-Content-Type-Options", "nosniff")
			if config.xFrameOptions != "" {
				ctx.SetHeader("X-Frame-Options", config.xFrameOptions)
			}
			if config.xXSSProtection != "" {
				ctx.SetHeader("X-XSS-Protection", config.xXSSProtection)
			}
			if config.referrerPolicy != "" {
				ctx.SetHeader("Referrer-Policy", config.referrerPolicy)
			}
			if config.csp != "" {
				ctx.SetHeader("Content-Security-Policy", config.csp)
			}
			ctx.SetHeader("Server", "")
			return next(ctx)
		}
	}
}

type hstsConfig struct {
	enabled           bool
	maxAge            time.Duration
	includeSubdomains bool
	preload           bool
}

func HSTSMiddleware(config hstsConfig) contracts.HTTPMiddleware {
	return func(next contracts.HTTPHandler) contracts.HTTPHandler {
		return func(ctx contracts.HTTPContext) error {
			if ctx.Request().TLS != nil && config.enabled {
				value := "max-age=" + strconv.FormatInt(int64(config.maxAge.Seconds()), 10)
				if config.includeSubdomains {
					value += "; includeSubDomains"
				}
				if config.preload {
					value += "; preload"
				}
				ctx.SetHeader("Strict-Transport-Security", value)
			}
			return next(ctx)
		}
	}
}
