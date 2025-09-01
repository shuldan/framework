package http

import (
	"encoding/json"

	"github.com/shuldan/framework/pkg/contracts"
)

type HTTPResponseImpl struct {
	statusCode int
	headers    map[string][]string
	body       []byte
	request    contracts.HTTPRequest
}

func (r *HTTPResponseImpl) StatusCode() int {
	return r.statusCode
}

func (r *HTTPResponseImpl) Header(key string) []string {
	return r.headers[key]
}

func (r *HTTPResponseImpl) Headers() map[string][]string {
	return r.headers
}

func (r *HTTPResponseImpl) Body() []byte {
	return r.body
}

func (r *HTTPResponseImpl) Request() contracts.HTTPRequest {
	return r.request
}

func (r *HTTPResponseImpl) JSON(v interface{}) error {
	return json.Unmarshal(r.body, v)
}

func (r *HTTPResponseImpl) String() string {
	return string(r.body)
}

func (r *HTTPResponseImpl) IsSuccess() bool {
	return r.statusCode >= 200 && r.statusCode < 300
}
