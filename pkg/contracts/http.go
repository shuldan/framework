package contracts

import (
	"context"
	"io"
	"net/http"
	"time"
)

type HTTPServer interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Addr() string
	Handler() http.Handler
}

type HTTPContext interface {
	HTTPRequestContext
	HTTPResponseWriter
	RequestContext

	FileUpload() HTTPFileUpload
	Streaming() HTTPStreamingContext
	Websocket() HTTPWebsocketContext
}

type HTTPRequestContext interface {
	Method() string
	Path() string
	Query(key string) string
	QueryDefault(key, defaultValue string) string
	QueryAll() map[string][]string
	Param(key string) string
	RequestHeader(key string) string
	Body() ([]byte, error)
	Request() *http.Request
}

type HTTPResponseWriter interface {
	SetHeader(key, value string) HTTPResponseWriter
	Status(code int) HTTPResponseWriter
	JSON(v interface{}) error
	String(s string) error
	Data(contentType string, data []byte) error
	Redirect(code int, location string) error
	NoContent() error
	StatusCode() int
}

type RequestContext interface {
	Context() context.Context
	SetContext(ctx context.Context)
	Set(key string, value interface{})
	Get(key string) (interface{}, bool)
	RequestID() string
	StartTime() time.Time
}

type HTTPRouter interface {
	GET(path string, handler HTTPHandler, middleware ...HTTPMiddleware)
	POST(path string, handler HTTPHandler, middleware ...HTTPMiddleware)
	PUT(path string, handler HTTPHandler, middleware ...HTTPMiddleware)
	DELETE(path string, handler HTTPHandler, middleware ...HTTPMiddleware)
	PATCH(path string, handler HTTPHandler, middleware ...HTTPMiddleware)
	HEAD(path string, handler HTTPHandler, middleware ...HTTPMiddleware)
	OPTIONS(path string, handler HTTPHandler, middleware ...HTTPMiddleware)

	Group(prefix string, middleware ...HTTPMiddleware) HTTPRouterGroup
	Use(middleware ...HTTPMiddleware)

	Static(path, root string)
	StaticFile(path, filepath string)

	Handle(method, path string, handler HTTPHandler, middleware ...HTTPMiddleware)
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

type HTTPRouterGroup interface {
	GET(path string, handler HTTPHandler, middleware ...HTTPMiddleware)
	POST(path string, handler HTTPHandler, middleware ...HTTPMiddleware)
	PUT(path string, handler HTTPHandler, middleware ...HTTPMiddleware)
	DELETE(path string, handler HTTPHandler, middleware ...HTTPMiddleware)
	PATCH(path string, handler HTTPHandler, middleware ...HTTPMiddleware)
	HEAD(path string, handler HTTPHandler, middleware ...HTTPMiddleware)
	OPTIONS(path string, handler HTTPHandler, middleware ...HTTPMiddleware)

	Group(prefix string, middleware ...HTTPMiddleware) HTTPRouterGroup
	Use(middleware ...HTTPMiddleware)
	Handle(method, path string, handler HTTPHandler, middleware ...HTTPMiddleware)
}

type HTTPHandler func(HTTPContext) error

type HTTPMiddleware func(HTTPHandler) HTTPHandler

type HTTPErrorHandler func(HTTPContext, error)

type HTTPClient interface {
	Get(ctx context.Context, url string, opts ...HTTPRequestOption) (HTTPResponse, error)
	Post(ctx context.Context, url string, body interface{}, opts ...HTTPRequestOption) (HTTPResponse, error)
	Put(ctx context.Context, url string, body interface{}, opts ...HTTPRequestOption) (HTTPResponse, error)
	Delete(ctx context.Context, url string, opts ...HTTPRequestOption) (HTTPResponse, error)
	Patch(ctx context.Context, url string, body interface{}, opts ...HTTPRequestOption) (HTTPResponse, error)
	Do(ctx context.Context, req HTTPRequest) (HTTPResponse, error)
}

type HTTPRequest interface {
	Method() string
	URL() string
	Header(key string) []string
	Headers() map[string][]string
	Body() []byte
	Context() context.Context
}

type HTTPResponse interface {
	StatusCode() int
	Header(key string) []string
	Headers() map[string][]string
	Body() []byte
	Request() HTTPRequest
	JSON(v interface{}) error
	String() string
	IsSuccess() bool
}

type HTTPRequestOption func(HTTPRequest)

type HTTPFileUpload interface {
	FormFile(name string) (HTTPFile, error)
	FormFiles(name string) ([]HTTPFile, error)
	FormValue(name string) string
	FormValues(name string) []string
	Parse(maxMemory int64) error
}

type HTTPFile interface {
	Header() map[string][]string
	Filename() string
	Size() int64
	Open() (io.ReadCloser, error)
	Save(path string) error
}

type HTTPStreamingContext interface {
	SetHeader(key, value string) HTTPStreamingContext
	SetContentType(contentType string) HTTPStreamingContext
	WriteChunk(data []byte) error
	WriteStringChunk(s string) error
	Flush()
	CloseNotify() <-chan struct{}
	IsClientClosed() bool
}

type HTTPWebsocketContext interface {
	IsWebsocket() bool
	Origin() string
	Subprotocols() []string
	Upgrade() (HTTPWebsocketConnection, error)
}

type HTTPWebsocketConnection interface {
	Read() <-chan HTTPWebsocketMessage
	Write(ctx context.Context, msg HTTPWebsocketMessage) error
	Close() error
	Ping(ctx context.Context) error
	IsClosed() bool
}

type HTTPWebsocketMessageType int

const (
	WebsocketText   HTTPWebsocketMessageType = 1
	WebsocketBinary HTTPWebsocketMessageType = 2
)

type HTTPWebsocketMessage struct {
	Type  HTTPWebsocketMessageType
	Data  []byte
	Error error
}
