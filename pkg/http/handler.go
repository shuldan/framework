package http

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/shuldan/framework/pkg/contracts"
	"github.com/shuldan/framework/pkg/errors"
)

type ErrorHandlerConfig struct {
	statusCodeMap  map[string]int
	userMessageMap map[string]string
	logLevel       string
	showStackTrace bool
	showDetails    bool
}

func NewErrorHandlerConfig() *ErrorHandlerConfig {
	return &ErrorHandlerConfig{
		statusCodeMap: map[string]int{
			string(errors.ErrValidation.Code):  http.StatusBadRequest,
			string(errors.ErrAuth.Code):        http.StatusUnauthorized,
			string(errors.ErrPermission.Code):  http.StatusForbidden,
			string(errors.ErrNotFound.Code):    http.StatusNotFound,
			string(errors.ErrConflict.Code):    http.StatusConflict,
			string(errors.ErrBusiness.Code):    http.StatusUnprocessableEntity,
			string(errors.ErrTimeout.Code):     http.StatusRequestTimeout,
			string(errors.ErrUnavailable.Code): http.StatusServiceUnavailable,
			string(errors.ErrInternal.Code):    http.StatusInternalServerError,
		},
		userMessageMap: map[string]string{
			string(errors.ErrValidation.Code):  "Invalid input data",
			string(errors.ErrAuth.Code):        "Authentication required",
			string(errors.ErrPermission.Code):  "Access denied",
			string(errors.ErrNotFound.Code):    "Resource not found",
			string(errors.ErrConflict.Code):    "Resource already exists",
			string(errors.ErrBusiness.Code):    "Operation not allowed",
			string(errors.ErrTimeout.Code):     "Request timeout",
			string(errors.ErrUnavailable.Code): "Service temporarily unavailable",
			string(errors.ErrInternal.Code):    "Internal server error",
		},
		logLevel:       "error",
		showStackTrace: false,
		showDetails:    false,
	}
}

func (c *ErrorHandlerConfig) WithStatusCode(errorCode string, status int) *ErrorHandlerConfig {
	c.statusCodeMap[errorCode] = status
	return c
}

func (c *ErrorHandlerConfig) WithUserMessage(errorCode, message string) *ErrorHandlerConfig {
	c.userMessageMap[errorCode] = message
	return c
}

func (c *ErrorHandlerConfig) WithLogLevel(level string) *ErrorHandlerConfig {
	c.logLevel = level
	return c
}

func (c *ErrorHandlerConfig) WithShowStackTrace(show bool) *ErrorHandlerConfig {
	c.showStackTrace = show
	return c
}

func (c *ErrorHandlerConfig) WithShowDetails(show bool) *ErrorHandlerConfig {
	c.showDetails = show
	return c
}

func (c *ErrorHandlerConfig) StatusCodeMap() map[string]int {
	return c.statusCodeMap
}

func (c *ErrorHandlerConfig) UserMessageMap() map[string]string {
	return c.userMessageMap
}

func (c *ErrorHandlerConfig) LogLevel() string {
	return c.logLevel
}

func (c *ErrorHandlerConfig) ShowStackTrace() bool {
	return c.showStackTrace
}

func (c *ErrorHandlerConfig) ShowDetails() bool {
	return c.showDetails
}

type errorHandler struct {
	config contracts.ErrorHandlerConfig
	logger contracts.Logger
}

func NewErrorHandler(config contracts.ErrorHandlerConfig, logger contracts.Logger) contracts.ErrorHandler {
	return &errorHandler{
		config: config,
		logger: logger,
	}
}

func (h *errorHandler) Handle(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	httpCtx := h.getHTTPContext(ctx)
	if httpCtx == nil {
		h.logger.Error("HTTP context not found in error handler", "error", err.Error())
		return err
	}
	if httpCtx.StatusCode() != 0 {
		h.logger.Warn("Response already sent, cannot handle error",
			"error", err.Error(),
			"status_code", httpCtx.StatusCode(),
			"request_id", httpCtx.RequestID())
		return nil
	}
	errorType, statusCode := h.determineErrorType(err)
	userMessage := h.getUserMessage(errorType)
	requestID := httpCtx.RequestID()
	h.logError(ctx, err, errorType, statusCode, requestID)
	httpCtx.SetHeader("Content-Type", "application/json; charset=utf-8")
	httpCtx.SetHeader("Cache-Control", "no-cache, no-store, must-revalidate")
	httpCtx.SetHeader("X-Request-ID", requestID)
	if statusCode == http.StatusUnauthorized {
		httpCtx.SetHeader("WWW-Authenticate", "Bearer")
	}
	if statusCode == http.StatusBadRequest {
		httpCtx.SetHeader("X-Validation-Error", "true")
	}
	httpCtx.Status(statusCode)
	response := map[string]interface{}{
		"error": map[string]interface{}{
			"code":       errorType,
			"message":    userMessage,
			"request_id": requestID,
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
		},
	}
	if h.config.ShowDetails() {
		if details := h.getErrorDetails(err); len(details) > 0 {
			response["error"].(map[string]interface{})["details"] = details
		}
	}
	if h.config.ShowStackTrace() && statusCode >= 500 {
		if stackTrace := h.getErrorStackTrace(err); stackTrace != "" {
			response["error"].(map[string]interface{})["stack_trace"] = stackTrace
		}
	}
	if statusCode == http.StatusTooManyRequests {
		httpCtx.SetHeader("Retry-After", "60")
	}
	if jsonErr := httpCtx.JSON(response); jsonErr != nil {
		h.logger.Error("Failed to send error response",
			"json_error", jsonErr.Error(),
			"original_error", err.Error(),
			"request_id", requestID)
		return jsonErr
	}
	return nil
}

func (h *errorHandler) determineErrorType(err error) (string, int) {
	if err == nil {
		return "", 0
	}

	code := errors.GetErrorCode(err)
	if code != "" {
		status := h.config.StatusCodeMap()[string(code)]
		if status == 0 {
			status = http.StatusInternalServerError
		}
		return string(code), status
	}

	if parts := strings.SplitN(err.Error(), ":", 2); len(parts) > 1 {
		errorCode := parts[0]
		status := h.config.StatusCodeMap()[errorCode]
		if status == 0 {
			status = http.StatusInternalServerError
		}
		return errorCode, status
	}

	return string(errors.ErrInternal.Code), http.StatusInternalServerError
}

func (h *errorHandler) getUserMessage(errorType string) string {
	if msg, ok := h.config.UserMessageMap()[errorType]; ok {
		return msg
	}
	return "An error occurred"
}

func (h *errorHandler) logError(_ context.Context, err error, errorType string, statusCode int, requestID string) {
	args := []any{
		"error", err.Error(),
		"error_type", errorType,
		"status_code", statusCode,
	}
	if requestID != "" {
		args = append(args, "request_id", requestID)
	}

	if strings.Contains(errorType, string(errors.ErrInternal.Code)) {
		var frameworkErr *errors.Error
		if errors.As(err, &frameworkErr) {
			args = append(args, "stack_trace", frameworkErr.Stack)
		}
	}

	switch {
	case statusCode >= 500:
		h.logger.Error("httpServer error", args...)
	case statusCode >= 400:
		h.logger.Warn("httpClient error", args...)
	default:
		h.logger.Info("Request error", args...)
	}
}

func (h *errorHandler) getHTTPContext(ctx context.Context) contracts.HTTPContext {
	if ctxValue := ctx.Value(ContextKey); ctxValue != nil {
		if hCtx, ok := ctxValue.(contracts.HTTPContext); ok {
			return hCtx
		}
	}
	return nil
}

func (h *errorHandler) getErrorDetails(err error) map[string]interface{} {
	var frameworkErr *errors.Error
	if errors.As(err, &frameworkErr) && len(frameworkErr.Details) > 0 {
		return frameworkErr.Details
	}
	return nil
}

func (h *errorHandler) getErrorStackTrace(err error) string {
	var frameworkErr *errors.Error
	if errors.As(err, &frameworkErr) && frameworkErr.Stack != "" {
		return frameworkErr.Stack
	}
	return ""
}
