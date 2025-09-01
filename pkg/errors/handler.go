package errors

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/shuldan/framework/pkg/contracts"
)

type httpContextKey struct{}

var HTTPContextKey = httpContextKey{}

type DefaultErrorHandlerConfig struct {
	statusCodeMap  map[string]int
	userMessageMap map[string]string
	logLevel       string
	showStackTrace bool
	showDetails    bool
}

func NewDefaultErrorHandlerConfig() *DefaultErrorHandlerConfig {
	return &DefaultErrorHandlerConfig{
		statusCodeMap: map[string]int{
			string(ErrValidation.Code):  http.StatusBadRequest,
			string(ErrAuth.Code):        http.StatusUnauthorized,
			string(ErrPermission.Code):  http.StatusForbidden,
			string(ErrNotFound.Code):    http.StatusNotFound,
			string(ErrConflict.Code):    http.StatusConflict,
			string(ErrBusiness.Code):    http.StatusUnprocessableEntity,
			string(ErrTimeout.Code):     http.StatusRequestTimeout,
			string(ErrUnavailable.Code): http.StatusServiceUnavailable,
			string(ErrInternal.Code):    http.StatusInternalServerError,
		},
		userMessageMap: map[string]string{
			string(ErrValidation.Code):  "Invalid input data",
			string(ErrAuth.Code):        "Authentication required",
			string(ErrPermission.Code):  "Access denied",
			string(ErrNotFound.Code):    "Resource not found",
			string(ErrConflict.Code):    "Resource already exists",
			string(ErrBusiness.Code):    "Operation not allowed",
			string(ErrTimeout.Code):     "Request timeout",
			string(ErrUnavailable.Code): "Service temporarily unavailable",
			string(ErrInternal.Code):    "Internal server error",
		},
		logLevel:       "error",
		showStackTrace: false,
		showDetails:    false,
	}
}

func (c *DefaultErrorHandlerConfig) StatusCodeMap() map[string]int {
	return c.statusCodeMap
}

func (c *DefaultErrorHandlerConfig) UserMessageMap() map[string]string {
	return c.userMessageMap
}

func (c *DefaultErrorHandlerConfig) LogLevel() string {
	return c.logLevel
}

func (c *DefaultErrorHandlerConfig) ShowStackTrace() bool {
	return c.showStackTrace
}

func (c *DefaultErrorHandlerConfig) ShowDetails() bool {
	return c.showDetails
}

func (c *DefaultErrorHandlerConfig) WithStatusCode(errorCode string, httpStatus int) *DefaultErrorHandlerConfig {
	c.statusCodeMap[errorCode] = httpStatus
	return c
}

func (c *DefaultErrorHandlerConfig) WithUserMessage(errorCode string, message string) *DefaultErrorHandlerConfig {
	c.userMessageMap[errorCode] = message
	return c
}

func (c *DefaultErrorHandlerConfig) WithLogLevel(level string) *DefaultErrorHandlerConfig {
	c.logLevel = level
	return c
}

func (c *DefaultErrorHandlerConfig) WithShowStackTrace(show bool) *DefaultErrorHandlerConfig {
	c.showStackTrace = show
	return c
}

func (c *DefaultErrorHandlerConfig) WithShowDetails(show bool) *DefaultErrorHandlerConfig {
	c.showDetails = show
	return c
}

type ErrorResponse struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	StackTrace string                 `json:"stack_trace,omitempty"`
	RequestID  string                 `json:"request_id,omitempty"`
}

type DefaultErrorHandler struct {
	config contracts.ErrorHandlerConfig
	logger contracts.Logger
}

func NewDefaultErrorHandler(config contracts.ErrorHandlerConfig, logger contracts.Logger) *DefaultErrorHandler {
	return &DefaultErrorHandler{
		config: config,
		logger: logger,
	}
}

func (h *DefaultErrorHandler) Handle(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	errorType, statusCode := h.determineErrorType(err)

	userMessage := h.getUserMessage(errorType)

	details := h.extractDetails(err)

	requestID := h.getRequestID(ctx)

	h.logError(ctx, err, errorType, statusCode, requestID)

	h.setHTTPStatus(ctx, statusCode)

	response := ErrorResponse{
		Code:      errorType,
		Message:   userMessage,
		Details:   details,
		RequestID: requestID,
	}

	if h.config.ShowStackTrace() {
		if frameworkErr, ok := err.(*Error); ok {
			response.StackTrace = frameworkErr.Stack
		}
	}

	return h.sendResponse(ctx, response)
}

func (h *DefaultErrorHandler) determineErrorType(err error) (string, int) {
	errorTypes := []*Error{
		ErrValidation,
		ErrAuth,
		ErrPermission,
		ErrNotFound,
		ErrConflict,
		ErrBusiness,
		ErrTimeout,
		ErrUnavailable,
	}

	for _, errorType := range errorTypes {
		if Is(err, errorType) {
			code := string(errorType.Code)
			statusCode := h.config.StatusCodeMap()[code]
			if statusCode == 0 {
				statusCode = http.StatusInternalServerError
			}
			return code, statusCode
		}
	}

	return string(ErrInternal.Code), http.StatusInternalServerError
}

func (h *DefaultErrorHandler) getUserMessage(errorType string) string {
	if message, exists := h.config.UserMessageMap()[errorType]; exists {
		return message
	}
	return "An error occurred"
}

func (h *DefaultErrorHandler) extractDetails(err error) map[string]interface{} {
	if frameworkErr, ok := err.(*Error); ok {
		if h.config.ShowDetails() && len(frameworkErr.Details) > 0 {
			return frameworkErr.Details
		}
	}
	return nil
}

func (h *DefaultErrorHandler) getRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return requestID
	}
	return ""
}

func (h *DefaultErrorHandler) logError(ctx context.Context, err error, errorType string, statusCode int, requestID string) {
	logArgs := []any{
		"error", err.Error(),
		"error_type", errorType,
		"status_code", statusCode,
	}

	if requestID != "" {
		logArgs = append(logArgs, "request_id", requestID)
	}

	if strings.Contains(errorType, string(ErrInternal.Code)) {
		var frameworkErr *Error
		if errors.As(err, &frameworkErr) {
			logArgs = append(logArgs, "stack_trace", frameworkErr.Stack)
		}
	}

	switch {
	case statusCode >= 500:
		h.logger.Error("Server error occurred", logArgs...)
	case statusCode >= 400:
		h.logger.Warn("Client error occurred", logArgs...)
	default:
		h.logger.Info("Request processed with error", logArgs...)
	}
}

func (h *DefaultErrorHandler) setHTTPStatus(ctx context.Context, statusCode int) {
	if httpCtx, ok := ctx.Value(HTTPContextKey).(contracts.HTTPContext); ok {
		httpCtx.Status(statusCode)
	}
}

func (h *DefaultErrorHandler) sendResponse(ctx context.Context, response ErrorResponse) error {
	if httpCtx, ok := ctx.Value(HTTPContextKey).(contracts.HTTPContext); ok {
		return httpCtx.JSON(response)
	}

	return &Error{
		Code:    Code(response.Code),
		Message: response.Message,
		Details: response.Details,
	}
}
