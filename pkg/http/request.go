package http

import (
	"context"
	"encoding/json"
)

type HTTPRequestImpl struct {
	method  string
	url     string
	headers map[string][]string
	body    []byte
	ctx     context.Context
}

func NewHTTPRequest(method, url string, body interface{}) *HTTPRequestImpl {
	req := &HTTPRequestImpl{
		method:  method,
		url:     url,
		headers: make(map[string][]string),
		ctx:     context.Background(),
	}

	if body != nil {
		switch v := body.(type) {
		case []byte:
			req.body = v
		case string:
			req.body = []byte(v)
		default:
			// Try to marshal as JSON
			if data, err := json.Marshal(body); err == nil {
				req.body = data
				req.headers["Content-Type"] = []string{"application/json"}
			}
		}
	}

	return req
}

func (r *HTTPRequestImpl) Method() string {
	return r.method
}

func (r *HTTPRequestImpl) URL() string {
	return r.url
}

func (r *HTTPRequestImpl) Header(key string) []string {
	return r.headers[key]
}

func (r *HTTPRequestImpl) Headers() map[string][]string {
	return r.headers
}

func (r *HTTPRequestImpl) Body() []byte {
	return r.body
}

func (r *HTTPRequestImpl) Context() context.Context {
	return r.ctx
}

func (r *HTTPRequestImpl) SetHeader(key string, values ...string) {
	r.headers[key] = values
}

func (r *HTTPRequestImpl) AddHeader(key, value string) {
	r.headers[key] = append(r.headers[key], value)
}

func (r *HTTPRequestImpl) SetContext(ctx context.Context) {
	r.ctx = ctx
}
