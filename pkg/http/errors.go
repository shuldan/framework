package http

import "github.com/shuldan/framework/pkg/errors"

var (
	ErrCodeGen = errors.WithPrefix("HTTP")

	ErrServerStart          = ErrCodeGen().New("failed to start server")
	ErrServerStop           = ErrCodeGen().New("failed to stop server")
	ErrServerAlreadyRunning = ErrCodeGen().New("server already running")
	ErrInvalidHandler       = ErrCodeGen().New("handler cannot be nil")
	ErrInvalidMiddleware    = ErrCodeGen().New("middleware cannot be nil")
	ErrRouteNotFound        = ErrCodeGen().New("route not found: {{.method}} {{.path}}")
	ErrMethodNotAllowed     = ErrCodeGen().New("method not allowed: {{.method}} {{.path}}")
	ErrInvalidPath          = ErrCodeGen().New("invalid path: {{.path}}")
	ErrBodyRead             = ErrCodeGen().New("failed to read request body")
	ErrJSONMarshal          = ErrCodeGen().New("failed to marshal JSON")
	ErrJSONUnmarshal        = ErrCodeGen().New("failed to unmarshal JSON")
	ErrResponseAlreadySent  = ErrCodeGen().New("response already sent")
	ErrFileNotFound         = ErrCodeGen().New("file not found: {{.path}}")
	ErrFormParse            = ErrCodeGen().New("failed to parse form")
	ErrWebsocketUpgrade     = ErrCodeGen().New("failed to upgrade to websocket")
	ErrWebsocketClosed      = ErrCodeGen().New("websocket connection closed")
	ErrHTTPRequest          = ErrCodeGen().New("HTTP request failed")
	ErrUnsupportedOpcode    = ErrCodeGen().New("unsupported opcode: {{.opcode}}")
	ErrMustCallParse        = ErrCodeGen().New("must call Parse() before accessing files")
)
