package http

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/shuldan/framework/pkg/contracts"
)

func TestServer(t *testing.T) {
	t.Parallel()
	logger := &mockLogger{}
	router := NewRouter(logger)
	router.GET("/health", func(ctx contracts.HTTPContext) error {
		return ctx.JSON(map[string]string{"status": "healthy"})
	})
	server, err := NewServer(":0", router, logger)
	if err != nil {
		t.Errorf("NewServer should not return an error: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := server.Start(ctx); err != nil {
		t.Fatalf("httpServer start failed: %v", err)
	}
	defer func(server contracts.HTTPServer, ctx context.Context) {
		err := server.Stop(ctx)
		if err != nil {
			t.Fatalf("httpServer stop failed: %v", err)
		}
	}(server, ctx)
	addr := server.Addr()
	if addr == "" {
		t.Fatal("httpServer address is empty")
	}
	resp, err := http.Get("http://" + addr + "/health")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			t.Fatalf("Body close failed: %v", err)
		}
	}(resp.Body)
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	if err := server.Stop(ctx); err != nil {
		t.Errorf("httpServer stop failed: %v", err)
	}
}

func TestServerAlreadyRunning(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	router := NewRouter(logger)
	server, err := NewServer("", router, logger)
	if err != nil {
		t.Errorf("NewServer should not return an error: %v", err)
	}

	ctx := context.Background()

	if err := server.Start(ctx); err != nil {
		t.Fatalf("First start failed: %v", err)
	}
	defer func(server contracts.HTTPServer, ctx context.Context) {
		err := server.Stop(ctx)
		if err != nil {
			t.Fatalf("First stop failed: %v", err)
		}
	}(server, ctx)

	if err := server.Start(ctx); !errors.Is(err, ErrServerAlreadyRunning) {
		t.Errorf("Expected ErrServerAlreadyRunning, got %v", err)
	}
}

func TestServerPanicOnNilDependencies(t *testing.T) {
	t.Parallel()
	_, err := NewServer("", nil, &mockLogger{})
	if err == nil {
		t.Errorf("Expected error for nil router")
	}
}

func TestClientCalculateRetryWait(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	config := ClientConfig{
		RetryWaitMin: time.Millisecond,
		RetryWaitMax: time.Second,
	}
	client := NewClientWithConfig(logger, config).(*httpClient)

	wait1 := client.calculateRetryWait(1)
	wait2 := client.calculateRetryWait(2)
	wait3 := client.calculateRetryWait(10)

	if wait1 < config.RetryWaitMin || wait1 > config.RetryWaitMax {
		t.Errorf("Wait time 1 out of bounds: %v", wait1)
	}

	if wait2 <= wait1 {
		t.Errorf("Wait time should increase: %v <= %v", wait2, wait1)
	}

	if wait3 > config.RetryWaitMax {
		t.Errorf("Wait time should be capped at max: %v > %v", wait3, config.RetryWaitMax)
	}
}
