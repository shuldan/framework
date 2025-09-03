package http

import (
	"encoding/json"
	"testing"
)

func TestHTTPRequest(t *testing.T) {
	t.Parallel()

	testData := map[string]string{"key": "value"}
	req := NewHTTPRequest("POST", "http://example.com", testData)

	if req.Method() != "POST" {
		t.Errorf("Expected method POST, got %s", req.Method())
	}
	if req.URL() != "http://example.com" {
		t.Errorf("Expected URL http://example.com, got %s", req.URL())
	}

	headers := req.Headers()
	if contentType := headers["Content-Type"]; len(contentType) == 0 || contentType[0] != contentTypeJSON {
		t.Error("Expected Content-Type application/json")
	}

	var decoded map[string]string
	if err := json.Unmarshal(req.Body(), &decoded); err != nil {
		t.Fatalf("Body decode failed: %v", err)
	}
	if decoded["key"] != "value" {
		t.Error("Body not encoded correctly")
	}
}

func TestHTTPRequestHeaders(t *testing.T) {
	t.Parallel()

	req := NewHTTPRequest("GET", "http://example.com", nil).(*httpRequest)
	req.SetHeader("Authorization", "Bearer token")
	req.AddHeader("Accept", "application/json")
	req.AddHeader("Accept", "text/plain")

	headers := req.Headers()
	if auth := headers["Authorization"]; len(auth) == 0 || auth[0] != "Bearer token" {
		t.Error("Authorization header not set correctly")
	}

	if accept := headers["Accept"]; len(accept) != 2 || accept[0] != "application/json" || accept[1] != "text/plain" {
		t.Error("Accept headers not added correctly")
	}
}
