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
	server, err := NewServer(router, logger, WithAddress(":0"))
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
	server, err := NewServer(router, logger, WithAddress(":0"))
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
	_, err := NewServer(nil, &mockLogger{})
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

func TestServerNilDependencies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		addr    string
		router  contracts.HTTPRouter
		logger  contracts.Logger
		wantErr bool
	}{
		{"Nil router", ":0", nil, &mockLogger{}, true},
		{"Nil logger", ":0", NewRouter(&mockLogger{}), nil, true},
		{"Valid dependencies", ":0", NewRouter(&mockLogger{}), &mockLogger{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewServer(tt.router, tt.logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewServer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServerDefaultAddress(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	router := NewRouter(logger)

	server, err := NewServer(router, logger)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	if server.Addr() != ":8080" {
		t.Errorf("Expected default address :8080, got %s", server.Addr())
	}
}

func TestServerStopBeforeStart(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	router := NewRouter(logger)
	server, err := NewServer(router, logger)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	ctx := context.Background()
	if err := server.Stop(ctx); err != nil {
		t.Errorf("Stop before start should not error: %v", err)
	}
}

func TestServerHandler(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	router := NewRouter(logger)
	server, err := NewServer(router, logger)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	handler := server.Handler()
	if handler != router {
		t.Error("Handler should return the router")
	}
}

func TestServerStartTimeout(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	router := NewRouter(logger)
	server, err := NewServer(router, logger, WithAddress(":0"))
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	if err := server.Start(ctx); err != nil {
		t.Fatalf("Server start failed: %v", err)
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stopCancel()

	if err := server.Stop(stopCtx); err != nil {
		t.Logf("Server stop with short timeout: %v", err)
	}
}

func TestServerConcurrentStartStop(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	router := NewRouter(logger)
	server, err := NewServer(router, logger, WithAddress(":0"))
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	ctx := context.Background()

	if err := server.Start(ctx); err != nil {
		t.Fatalf("First start failed: %v", err)
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		if err := server.Stop(ctx); err != nil {
			t.Errorf("Concurrent stop failed: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	if err := server.Start(ctx); err != nil {
		t.Errorf("Restart after stop failed: %v", err)
	}

	if err := server.Stop(ctx); err != nil {
		t.Errorf("Final stop failed: %v", err)
	}
}

func TestServerInvalidAddress(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	router := NewRouter(logger)
	server, err := NewServer(router, logger, WithAddress("invalid-address"))
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	ctx := context.Background()
	if err := server.Start(ctx); err == nil {
		t.Error("Expected error for invalid address")
	}
}
