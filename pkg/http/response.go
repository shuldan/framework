package http

import (
	"encoding/json"

	"github.com/shuldan/framework/pkg/contracts"
)

type httpResponse struct {
	statusCode int
	headers    map[string][]string
	body       []byte
	request    contracts.HTTPRequest
}

func (r *httpResponse) StatusCode() int {
	return r.statusCode
}

func (r *httpResponse) Header(key string) []string {
	return r.headers[key]
}

func (r *httpResponse) Headers() map[string][]string {
	return r.headers
}

func (r *httpResponse) Body() []byte {
	return r.body
}

func (r *httpResponse) Request() contracts.HTTPRequest {
	return r.request
}

func (r *httpResponse) JSON(v interface{}) error {
	return json.Unmarshal(r.body, v)
}

func (r *httpResponse) String() string {
	return string(r.body)
}

func (r *httpResponse) IsSuccess() bool {
	return r.statusCode >= 200 && r.statusCode < 300
}
