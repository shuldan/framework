package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shuldan/framework/pkg/contracts"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	client := NewClient(logger).(*httpClient)

	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.client == nil {
		t.Error("HTTP client is nil")
	}
	if client.logger != logger {
		t.Error("Logger not set correctly")
	}
	if client.client.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", client.client.Timeout)
	}
}

func TestNewClientWithConfig(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	config := ClientConfig{
		Timeout:      10 * time.Second,
		MaxRetries:   5,
		RetryWaitMin: 2 * time.Second,
		RetryWaitMax: 20 * time.Second,
	}

	client := NewClientWithConfig(logger, config).(*httpClient)

	if client == nil {
		t.Fatal("NewClientWithConfig returned nil")
	}
	if client.client.Timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", client.client.Timeout)
	}
	if client.config.MaxRetries != 5 {
		t.Errorf("Expected MaxRetries 5, got %d", client.config.MaxRetries)
	}
}

func TestClientHTTPMethods(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := map[string]string{
			"method": r.Method,
			"path":   r.URL.Path,
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	logger := &mockLogger{}
	client := NewClient(logger)

	tests := []struct {
		name   string
		method func(context.Context, string, ...contracts.HTTPRequestOption) (contracts.HTTPResponse, error)
		want   string
	}{
		{"GET", func(ctx context.Context, url string, opts ...contracts.HTTPRequestOption) (contracts.HTTPResponse, error) {
			return client.Get(ctx, url, opts...)
		}, "GET"},
		{"DELETE", func(ctx context.Context, url string, opts ...contracts.HTTPRequestOption) (contracts.HTTPResponse, error) {
			return client.Delete(ctx, url, opts...)
		}, "DELETE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := tt.method(context.Background(), server.URL+"/test")
			if err != nil {
				t.Fatalf("HTTP request failed: %v", err)
			}
			if resp.StatusCode() != 200 {
				t.Errorf("Expected status 200, got %d", resp.StatusCode())
			}

			var result map[string]string
			if err := resp.JSON(&result); err != nil {
				t.Fatalf("JSON decode failed: %v", err)
			}
			if result["method"] != tt.want {
				t.Errorf("Expected method %s, got %s", tt.want, result["method"])
			}
		})
	}
}

func TestClientWithBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(map[string]interface{}{
			"method":      r.Method,
			"body":        string(body),
			"contentType": r.Header.Get("Content-Type"),
		})
		if err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	logger := &mockLogger{}
	client := NewClient(logger)

	testData := map[string]string{"key": "value"}

	resp, err := client.Post(context.Background(), server.URL, testData)
	if err != nil {
		t.Fatalf("POST request failed: %v", err)
	}

	var result map[string]interface{}
	if err := resp.JSON(&result); err != nil {
		t.Fatalf("JSON decode failed: %v", err)
	}

	if result["method"] != "POST" {
		t.Errorf("Expected method POST, got %v", result["method"])
	}
	if result["contentType"] != contentTypeJSON {
		t.Errorf("Expected content type application/json, got %v", result["contentType"])
	}
}

func TestClientRetry(t *testing.T) {
	t.Parallel()
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(map[string]int{"attempts": attempts})
		if err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	logger := &mockLogger{}
	config := ClientConfig{
		MaxRetries:   3,
		RetryWaitMin: 100 * time.Microsecond,
		RetryWaitMax: 500 * time.Microsecond,
		RetryCondition: func(resp contracts.HTTPResponse, err error) bool {
			return resp != nil && resp.StatusCode() >= 500
		},
	}
	client := NewClientWithConfig(logger, config)

	resp, err := client.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode() != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode())
	}
	var result map[string]int
	if err := resp.JSON(&result); err != nil {
		t.Fatalf("JSON decode failed: %v", err)
	}
	if result["attempts"] != 3 {
		t.Errorf("Expected 3 attempts, got %d", result["attempts"])
	}
}

func TestClientTimeout(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := &mockLogger{}
	config := ClientConfig{
		Timeout:    1 * time.Millisecond,
		MaxRetries: 0,
	}
	client := NewClientWithConfig(logger, config)

	_, err := client.Get(context.Background(), server.URL)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}
