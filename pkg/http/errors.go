package http

import "github.com/shuldan/framework/pkg/errors"

var (
	ErrCodeGen = errors.WithPrefix("HTTP")

	ErrServerStart               = ErrCodeGen().New("failed to start server")
	ErrServerStop                = ErrCodeGen().New("failed to stop server")
	ErrServerAlreadyRunning      = ErrCodeGen().New("server already running")
	ErrInvalidHandler            = ErrCodeGen().New("handler cannot be nil")
	ErrRouteNotFound             = ErrCodeGen().New("route not found: {{.method}} {{.path}}")
	ErrMethodNotAllowed          = ErrCodeGen().New("method not allowed: {{.method}} {{.path}}")
	ErrBodyRead                  = ErrCodeGen().New("failed to read request body")
	ErrJSONMarshal               = ErrCodeGen().New("failed to marshal JSON")
	ErrResponseAlreadySent       = ErrCodeGen().New("response already sent")
	ErrFileNotFound              = ErrCodeGen().New("file not found: {{.path}}")
	ErrFormParse                 = ErrCodeGen().New("failed to parse form")
	ErrWebsocketUpgrade          = ErrCodeGen().New("failed to upgrade to websocket")
	ErrWebsocketClosed           = ErrCodeGen().New("websocket connection closed")
	ErrHTTPRequest               = ErrCodeGen().New("HTTP request failed")
	ErrUnsupportedOpcode         = ErrCodeGen().New("unsupported opcode: {{.opcode}}")
	ErrLoggerNotFound            = ErrCodeGen().New("logger not found")
	ErrInvalidLoggerInstance     = ErrCodeGen().New("invalid logger")
	ErrHTTPRouterNotFound        = ErrCodeGen().New("http router not found")
	ErrInvalidHTTPRouterInstance = ErrCodeGen().New("invalid http router")
	ErrHTTPServerNotFound        = ErrCodeGen().New("http server not found")
	ErrInvalidHTTPServerInstance = ErrCodeGen().New("invalid http server")
)
