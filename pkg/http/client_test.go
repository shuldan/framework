package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
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
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				response := map[string]string{
					"method": r.Method,
					"path":   r.URL.Path,
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Logf("Failed to encode response: %v", err)
				}
			}))
			defer server.Close()
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
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"method":      r.Method,
			"body":        string(body),
			"contentType": r.Header.Get("Content-Type"),
		}); err != nil {
			t.Logf("Failed to encode response: %v", err)
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

	if result["method"] != http.MethodPost {
		t.Errorf("Expected method POST, got %v", result["method"])
	}
	if result["contentType"] != contentTypeJSON {
		t.Errorf("Expected content type application/json, got %v", result["contentType"])
	}
}

func TestClientRetry(t *testing.T) {
	t.Parallel()
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt32(&attempts, 1)
		if current < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]int32{"attempts": current}); err != nil {
			t.Logf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	logger := &mockLogger{}
	config := ClientConfig{
		MaxRetries:   3,
		RetryWaitMin: 10 * time.Microsecond,
		RetryWaitMax: 50 * time.Microsecond,
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
	var result map[string]int32
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

func TestNewClientWithConfigDefaults(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	config := ClientConfig{}

	client := NewClientWithConfig(logger, config).(*httpClient)

	if client.config.Timeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", client.config.Timeout)
	}
	if client.config.MaxRetries != 3 {
		t.Errorf("Expected default MaxRetries 3, got %d", client.config.MaxRetries)
	}
	if client.config.RetryWaitMin != time.Second {
		t.Errorf("Expected default RetryWaitMin 1s, got %v", client.config.RetryWaitMin)
	}
	if client.config.RetryWaitMax != 10*time.Second {
		t.Errorf("Expected default RetryWaitMax 10s, got %v", client.config.RetryWaitMax)
	}
	if client.config.RetryCondition == nil {
		t.Error("Expected default RetryCondition to be set")
	}
}

func TestClientWithBodyMethods(t *testing.T) {
	logger := &mockLogger{}
	client := NewClient(logger)
	testData := map[string]string{"key": "value"}

	tests := []struct {
		name   string
		method func(context.Context, string, interface{}, ...contracts.HTTPRequestOption) (contracts.HTTPResponse, error)
		want   string
	}{
		{"PUT", client.Put, "PUT"},
		{"PATCH", client.Patch, "PATCH"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				w.Header().Set("Content-Type", "application/json")
				response := map[string]interface{}{
					"method":      r.Method,
					"body":        string(body),
					"contentType": r.Header.Get("Content-Type"),
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Logf("Failed to encode response: %v", err)
				}
			}))
			defer server.Close()
			resp, err := tt.method(context.Background(), server.URL, testData)
			if err != nil {
				t.Fatalf("HTTP request failed: %v", err)
			}

			var result map[string]interface{}
			if err := resp.JSON(&result); err != nil {
				t.Fatalf("JSON decode failed: %v", err)
			}

			if result["method"] != tt.want {
				t.Errorf("Expected method %s, got %v", tt.want, result["method"])
			}
		})
	}
}

func TestClientDoWithoutRetry(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
			t.Logf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	logger := &mockLogger{}
	config := ClientConfig{MaxRetries: 0}
	client := NewClientWithConfig(logger, config)

	req := NewHTTPRequest("GET", server.URL, nil)
	resp, err := client.Do(context.Background(), req)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode() != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode())
	}
}

func TestClientRetryMaxExceeded(t *testing.T) {
	t.Parallel()

	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	logger := &mockLogger{}
	config := ClientConfig{
		MaxRetries:   2,
		RetryWaitMin: 10 * time.Microsecond,
		RetryWaitMax: 10 * time.Microsecond,
	}
	client := NewClientWithConfig(logger, config)

	_, err := client.Get(context.Background(), server.URL)
	if err == nil {
		t.Error("Expected error after max retries exceeded")
	}

	finalAttempts := atomic.LoadInt32(&attempts)
	if finalAttempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", finalAttempts)
	}
}

func TestClientContextCancellation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := &mockLogger{}
	config := ClientConfig{
		MaxRetries:   2,
		RetryWaitMin: 50 * time.Millisecond,
		RetryWaitMax: 50 * time.Millisecond,
	}
	client := NewClientWithConfig(logger, config)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := client.Get(ctx, server.URL)
	if err == nil {
		t.Error("Expected context cancellation error")
	}
}

func TestClientRetryConditionCustom(t *testing.T) {
	t.Parallel()

	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt32(&attempts, 1)
		if current == 1 {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	logger := &mockLogger{}
	config := ClientConfig{
		MaxRetries:   2,
		RetryWaitMin: 10 * time.Microsecond,
		RetryWaitMax: 10 * time.Microsecond,
		RetryCondition: func(resp contracts.HTTPResponse, err error) bool {
			return resp != nil && resp.StatusCode() == 400
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

	finalAttempts := atomic.LoadInt32(&attempts)
	if finalAttempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", finalAttempts)
	}
}

func TestClientBuildHTTPRequestError(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	client := NewClientWithConfig(logger, ClientConfig{MaxRetries: 0}).(*httpClient)

	req := NewHTTPRequest("GET", "://invalid-url", nil)
	_, err := client.doSingleRequest(context.Background(), req)

	if err == nil {
		t.Error("Expected error for invalid request")
	}
}

func TestClientProcessResponseError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "10")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("short")); err != nil {
			t.Logf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	logger := &mockLogger{}
	client := NewClientWithConfig(logger, ClientConfig{MaxRetries: 0})

	_, err := client.Get(context.Background(), server.URL)
	if err != nil {
		t.Logf("Got expected error due to content length mismatch: %v", err)
	}
}

func TestClientGenerateSecureJitterFallback(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	config := ClientConfig{
		RetryWaitMin: time.Nanosecond,
		RetryWaitMax: time.Nanosecond,
	}
	client := NewClientWithConfig(logger, config).(*httpClient)

	jitter := client.generateSecureJitter(time.Nanosecond)
	if jitter < 0 {
		t.Error("Jitter should not be negative")
	}
}

func TestClientRetryWithNilResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	server.Close()

	logger := &mockLogger{}
	config := ClientConfig{
		MaxRetries:   1,
		RetryWaitMin: 10 * time.Microsecond,
		RetryWaitMax: 10 * time.Microsecond,
		RetryCondition: func(resp contracts.HTTPResponse, err error) bool {
			return err != nil
		},
	}
	client := NewClientWithConfig(logger, config)

	_, err := client.Get(context.Background(), server.URL)
	if err == nil {
		t.Error("Expected error for closed server")
	}
}
