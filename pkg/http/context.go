package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/shuldan/framework/pkg/contracts"
)

type httpContext struct {
	req          *http.Request
	resp         http.ResponseWriter
	logger       contracts.Logger
	statusCode   int
	responseSent bool
	params       map[string]interface{}
	body         []byte
	bodyRead     bool
	startTime    time.Time
	requestID    string
	mu           sync.RWMutex
}

func NewHTTPContext(w http.ResponseWriter, r *http.Request, logger contracts.Logger) contracts.HTTPContext {
	var requestID string
	if clientReqID := r.Header.Get("X-Request-ID"); clientReqID != "" {
		requestID = clientReqID
	} else {
		requestID = generateRequestID()
	}

	ctx := context.WithValue(r.Context(), RequestID, requestID)
	r = r.WithContext(ctx)

	return &httpContext{
		req:       r,
		resp:      w,
		logger:    logger,
		params:    make(map[string]interface{}),
		startTime: time.Now(),
		requestID: requestID,
	}
}

func (c *httpContext) Context() context.Context {
	return c.req.Context()
}

func (c *httpContext) SetContext(ctx context.Context) {
	if oldReqID := c.req.Context().Value(RequestID); oldReqID != nil {
		ctx = context.WithValue(ctx, RequestID, oldReqID)
	}
	if oldHTTPCtx := c.req.Context().Value(ContextKey); oldHTTPCtx != nil {
		ctx = context.WithValue(ctx, ContextKey, oldHTTPCtx)
	}
	c.req = c.req.WithContext(ctx)
}

func (c *httpContext) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.params[key] = value
}

func (c *httpContext) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	value, exists := c.params[key]
	return value, exists
}

func (c *httpContext) RequestID() string {
	return c.requestID
}

func (c *httpContext) StartTime() time.Time {
	return c.startTime
}

func (c *httpContext) Method() string {
	return c.req.Method
}

func (c *httpContext) Path() string {
	return c.req.URL.Path
}

func (c *httpContext) Query(key string) string {
	return c.req.URL.Query().Get(key)
}

func (c *httpContext) QueryDefault(key, defaultValue string) string {
	if value := c.Query(key); value != "" {
		return value
	}
	return defaultValue
}

func (c *httpContext) QueryAll() map[string][]string {
	return c.req.URL.Query()
}

func (c *httpContext) Param(key string) string {
	if value, exists := c.params[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

func (c *httpContext) RequestHeader(key string) string {
	return c.req.Header.Get(key)
}

func (c *httpContext) Body() ([]byte, error) {
	if !c.bodyRead {
		var err error
		c.body, err = io.ReadAll(c.req.Body)
		if err != nil {
			return nil, ErrBodyRead.WithCause(err)
		}
		c.bodyRead = true
		if closeErr := c.req.Body.Close(); closeErr != nil && c.logger != nil {
			c.logger.Error("Failed to close request body", "error", closeErr)
		}
	}
	return c.body, nil
}

func (c *httpContext) Request() *http.Request {
	return c.req
}

func (c *httpContext) SetHeader(key, value string) contracts.HTTPResponseWriter {
	c.resp.Header().Set(key, value)
	return c
}

func (c *httpContext) Status(code int) contracts.HTTPResponseWriter {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.statusCode = code
	return c
}

func (c *httpContext) JSON(v interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.responseSent {
		return ErrResponseAlreadySent
	}

	data, err := json.Marshal(v)
	if err != nil {
		return ErrJSONMarshal.WithCause(err)
	}

	c.resp.Header().Set("Content-Type", "application/json")

	if c.statusCode == 0 {
		c.statusCode = http.StatusOK
	}

	c.resp.WriteHeader(c.statusCode)
	_, err = c.resp.Write(data)
	c.responseSent = true

	return err
}

func (c *httpContext) String(s string) error {
	if c.responseSent {
		return ErrResponseAlreadySent
	}

	c.SetHeader("Content-Type", "text/plain; charset=utf-8")

	if c.statusCode == 0 {
		c.statusCode = http.StatusOK
	}

	c.resp.WriteHeader(c.statusCode)
	_, err := c.resp.Write([]byte(s))
	c.responseSent = true

	return err
}

func (c *httpContext) Data(contentType string, data []byte) error {
	if c.responseSent {
		return ErrResponseAlreadySent
	}

	c.SetHeader("Content-Type", contentType)

	if c.statusCode == 0 {
		c.statusCode = http.StatusOK
	}

	c.resp.WriteHeader(c.statusCode)
	_, err := c.resp.Write(data)
	c.responseSent = true

	return err
}

func (c *httpContext) Redirect(code int, location string) error {
	if c.responseSent {
		return ErrResponseAlreadySent
	}

	c.SetHeader("Location", location)
	c.resp.WriteHeader(code)
	c.responseSent = true

	return nil
}

func (c *httpContext) NoContent() error {
	if c.responseSent {
		return ErrResponseAlreadySent
	}

	c.resp.WriteHeader(http.StatusNoContent)
	c.responseSent = true

	return nil
}

func (c *httpContext) StatusCode() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.statusCode
}

func (c *httpContext) FileUpload() contracts.HTTPFileUpload {
	return &FileUpload{ctx: c, logger: c.logger}
}

func (c *httpContext) Streaming() contracts.HTTPStreamingContext {
	return &StreamingContext{ctx: c}
}

func (c *httpContext) Websocket() contracts.HTTPWebsocketContext {
	return &WebsocketContext{ctx: c, logger: c.logger}
}

func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}
