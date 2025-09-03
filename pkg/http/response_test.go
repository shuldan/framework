package http

import (
	"encoding/json"
	"testing"
)

func TestHTTPResponse(t *testing.T) {
	t.Parallel()

	req := NewHTTPRequest("GET", "http://example.com", nil)
	headers := make(map[string][]string)
	headers["Content-Type"] = []string{"application/json"}

	testData := map[string]string{"message": "success"}
	body, _ := json.Marshal(testData)

	resp := &httpResponse{
		statusCode: 200,
		headers:    headers,
		body:       body,
		request:    req,
	}

	if resp.StatusCode() != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode())
	}

	if !resp.IsSuccess() {
		t.Error("Expected IsSuccess to be true")
	}

	if contentType := resp.Header("Content-Type"); len(contentType) == 0 || contentType[0] != "application/json" {
		t.Error("Content-Type header not returned correctly")
	}

	var decoded map[string]string
	if err := resp.JSON(&decoded); err != nil {
		t.Fatalf("JSON decode failed: %v", err)
	}

	if decoded["message"] != "success" {
		t.Error("JSON not decoded correctly")
	}

	if resp.String() != string(body) {
		t.Error("String() not returned correctly")
	}

	if resp.Request() != req {
		t.Error("Request not returned correctly")
	}
}
