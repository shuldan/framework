package http

import "testing"

func TestOptions(t *testing.T) {
	t.Parallel()

	t.Run("WithHeader", func(t *testing.T) {
		req := NewHTTPRequest("GET", "http://example.com", nil)
		opt := WithHeader("Authorization", "Bearer token")
		opt(req)

		headers := req.Headers()
		if auth := headers["Authorization"]; len(auth) == 0 || auth[0] != "Bearer token" {
			t.Error("WithHeader option not applied correctly")
		}
	})

	t.Run("WithBearerToken", func(t *testing.T) {
		req := NewHTTPRequest("GET", "http://example.com", nil)
		opt := WithBearerToken("test-token")
		opt(req)

		headers := req.Headers()
		if auth := headers["Authorization"]; len(auth) == 0 || auth[0] != "Bearer test-token" {
			t.Error("WithBearerToken option not applied correctly")
		}
	})
}
