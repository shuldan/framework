package http

import (
	"context"
	"encoding/json"

	"github.com/shuldan/framework/pkg/contracts"
)

type httpRequest struct {
	method  string
	url     string
	headers map[string][]string
	body    []byte
	ctx     context.Context
}

func NewHTTPRequest(method, url string, body interface{}) contracts.HTTPRequest {
	req := &httpRequest{
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
			if data, err := json.Marshal(body); err == nil {
				req.body = data
				req.headers["Content-Type"] = []string{"application/json"}
			}
		}
	}

	return req
}

func (r *httpRequest) Method() string {
	return r.method
}

func (r *httpRequest) URL() string {
	return r.url
}

func (r *httpRequest) Header(key string) []string {
	return r.headers[key]
}

func (r *httpRequest) Headers() map[string][]string {
	return r.headers
}

func (r *httpRequest) Body() []byte {
	return r.body
}

func (r *httpRequest) Context() context.Context {
	return r.ctx
}

func (r *httpRequest) SetHeader(key string, values ...string) {
	r.headers[key] = values
}

func (r *httpRequest) AddHeader(key, value string) {
	r.headers[key] = append(r.headers[key], value)
}

func (r *httpRequest) SetContext(ctx context.Context) {
	r.ctx = ctx
}
