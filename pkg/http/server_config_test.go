package http

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/shuldan/framework/pkg/contracts"
)

func TestServerAppliesConfiguredTimeouts(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	router := NewRouter(logger)

	slowHandler := func(ctx contracts.HTTPContext) error {
		time.Sleep(150 * time.Millisecond)
		return ctx.JSON(map[string]string{"status": "ok"})
	}
	router.GET("/slow", slowHandler)

	server, err := NewServer(router, logger,
		WithAddress(":0"),
		WithReadTimeout(100*time.Millisecond),
		WithWriteTimeout(100*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	ctx := context.Background()
	if err := server.Start(ctx); err != nil {
		t.Fatalf("Server start failed: %v", err)
	}
	defer func() {
		if err := server.Stop(ctx); err != nil {
			t.Logf("Server stop failed: %v", err)
		}
	}()

	httpSrv := server.(*httpServer)
	if httpSrv.server.ReadTimeout != 100*time.Millisecond {
		t.Errorf("ReadTimeout not applied: got %v", httpSrv.server.ReadTimeout)
	}
	if httpSrv.server.WriteTimeout != 100*time.Millisecond {
		t.Errorf("WriteTimeout not applied: got %v", httpSrv.server.WriteTimeout)
	}

	client := &http.Client{Timeout: 500 * time.Millisecond}
	_, err = client.Get("http://" + server.Addr() + "/slow")
	if err == nil {
		t.Error("Expected timeout error for slow endpoint")
	}
}

func TestServerAppliesIdleTimeout(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	router := NewRouter(logger)
	router.GET("/test", func(ctx contracts.HTTPContext) error {
		return ctx.JSON(map[string]string{"status": "ok"})
	})

	server, err := NewServer(router, logger,
		WithAddress(":0"),
		WithIdleTimeout(50*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	ctx := context.Background()
	if err := server.Start(ctx); err != nil {
		t.Fatalf("Server start failed: %v", err)
	}
	defer func() {
		if err := server.Stop(ctx); err != nil {
			t.Logf("Server stop failed: %v", err)
		}
	}()

	httpSrv := server.(*httpServer)
	if httpSrv.server.IdleTimeout != 50*time.Millisecond {
		t.Errorf("IdleTimeout not applied: got %v", httpSrv.server.IdleTimeout)
	}

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:    1,
			IdleConnTimeout: 10 * time.Millisecond,
		},
	}

	resp, err := client.Get("http://" + server.Addr() + "/test")
	if err != nil {
		t.Fatalf("First request failed: %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Errorf("First request close: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	resp, err = client.Get("http://" + server.Addr() + "/test")
	if err != nil {
		t.Fatalf("Second request failed: %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Errorf("Second request close: %v", err)
	}
}

func TestServerAppliesReadHeaderTimeout(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	router := NewRouter(logger)
	router.GET("/test", func(ctx contracts.HTTPContext) error {
		return ctx.JSON(map[string]string{"status": "ok"})
	})

	server, err := NewServer(router, logger,
		WithAddress(":0"),
		WithReadHeaderTimeout(100*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	ctx := context.Background()
	if err := server.Start(ctx); err != nil {
		t.Fatalf("Server start failed: %v", err)
	}
	defer func() {
		if err := server.Stop(ctx); err != nil {
			t.Logf("Server stop failed: %v", err)
		}
	}()

	httpSrv := server.(*httpServer)
	if httpSrv.server.ReadHeaderTimeout != 100*time.Millisecond {
		t.Errorf("ReadHeaderTimeout not applied: got %v", httpSrv.server.ReadHeaderTimeout)
	}
}

func TestServerAppliesShutdownTimeout(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	router := NewRouter(logger)

	requestStarted := make(chan struct{})
	router.GET("/long", func(ctx contracts.HTTPContext) error {
		close(requestStarted)
		select {
		case <-time.After(500 * time.Millisecond):
			return ctx.JSON(map[string]string{"status": "completed"})
		case <-ctx.ParentContext().Done():
			return ctx.ParentContext().Err()
		}
	})

	server, err := NewServer(router, logger,
		WithAddress(":0"),
		WithShutdownTimeout(100*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	ctx := context.Background()
	if err := server.Start(ctx); err != nil {
		t.Fatalf("Server start failed: %v", err)
	}

	go func() {
		_, _ = http.Get("http://" + server.Addr() + "/long")
	}()
	<-requestStarted

	httpSrv := server.(*httpServer)
	if httpSrv.config.shutdownTimeout != 100*time.Millisecond {
		t.Errorf("ShutdownTimeout not applied in config: got %v", httpSrv.config.shutdownTimeout)
	}

	startShutdown := time.Now()
	err = server.Stop(ctx)
	shutdownDuration := time.Since(startShutdown)

	if err == nil && shutdownDuration > 150*time.Millisecond {
		t.Error("Shutdown took too long, timeout might not be applied")
	}
	if shutdownDuration < 90*time.Millisecond {
		t.Error("Shutdown too fast, timeout might not be working")
	}
}

func testHelperStartServer(t *testing.T, router contracts.HTTPRouter, logger contracts.Logger, options ...ServerOption) (contracts.HTTPServer, context.Context, context.CancelFunc) {
	t.Helper()

	server, err := NewServer(router, logger, options...)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	if err := server.Start(ctx); err != nil {
		cancel()
		if err := server.Stop(ctx); err != nil {
			t.Logf("Failed to stop server after start error: %v", err)
		}
		t.Fatalf("Server start failed: %v", err)
	}

	t.Cleanup(func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer stopCancel()
		if err := server.Stop(stopCtx); err != nil {
			t.Logf("Server stop failed during cleanup: %v", err)
		}
		cancel()
	})

	return server, ctx, cancel
}

func TestServerAppliesAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		configuredAddr string
		expectedAddr   string
	}{
		{
			name:           "Specific port",
			configuredAddr: ":8765",
			expectedAddr:   "[::]:8765",
		},
		{
			name:           "Localhost with port",
			configuredAddr: "localhost:8766",
			expectedAddr:   "127.0.0.1:8766",
		},
		{
			name:           "Dynamic port",
			configuredAddr: ":0",
			expectedAddr:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := &mockLogger{}
			router := NewRouter(logger)
			router.GET("/ping", func(ctx contracts.HTTPContext) error {
				return ctx.JSON(map[string]string{"pong": "ok"})
			})

			server, _, _ := testHelperStartServer(t, router, logger, WithAddress(tt.configuredAddr))
			httpSrv := server.(*httpServer)
			if httpSrv.config.address != tt.configuredAddr {
				t.Errorf("Address not stored in config: got %s, want %s",
					httpSrv.config.address, tt.configuredAddr)
			}
			actualAddr := server.Addr()
			if tt.configuredAddr == ":0" {
				if actualAddr == ":0" || actualAddr == "" {
					t.Error("Dynamic port not assigned")
				}
			} else {
				if actualAddr != tt.expectedAddr {
					t.Errorf("Expected server.Addr() = %q, got %q", tt.expectedAddr, actualAddr)
				}
			}
		})
	}
}

func TestServerConfigurationOverrides(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	router := NewRouter(logger)

	defaultConfig := defaultServerConfig()
	customOptions := []ServerOption{
		WithAddress(":9999"),
		WithReadTimeout(15 * time.Second),
		WithWriteTimeout(25 * time.Second),
		WithIdleTimeout(35 * time.Second),
		WithShutdownTimeout(45 * time.Second),
		WithReadHeaderTimeout(55 * time.Second),
	}

	server, err := NewServer(router, logger, customOptions...)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	httpSrv := server.(*httpServer)

	if httpSrv.config.address == defaultConfig.address {
		t.Error("Address not overridden from default")
	}
	if httpSrv.config.readTimeout == defaultConfig.readTimeout {
		t.Error("ReadTimeout not overridden from default")
	}
	if httpSrv.config.writeTimeout == defaultConfig.writeTimeout {
		t.Error("WriteTimeout not overridden from default")
	}
	if httpSrv.config.idleTimeout == defaultConfig.idleTimeout {
		t.Error("IdleTimeout not overridden from default")
	}
	if httpSrv.config.shutdownTimeout == defaultConfig.shutdownTimeout {
		t.Error("ShutdownTimeout not overridden from default")
	}
	if httpSrv.config.readHeaderTimeout == defaultConfig.readHeaderTimeout {
		t.Error("ReadHeaderTimeout not overridden from default")
	}

	if httpSrv.config.address != ":9999" {
		t.Errorf("Address not applied: got %s", httpSrv.config.address)
	}
	if httpSrv.config.readTimeout != 15*time.Second {
		t.Errorf("ReadTimeout not applied: got %v", httpSrv.config.readTimeout)
	}
	if httpSrv.config.writeTimeout != 25*time.Second {
		t.Errorf("WriteTimeout not applied: got %v", httpSrv.config.writeTimeout)
	}
	if httpSrv.config.idleTimeout != 35*time.Second {
		t.Errorf("IdleTimeout not applied: got %v", httpSrv.config.idleTimeout)
	}
	if httpSrv.config.shutdownTimeout != 45*time.Second {
		t.Errorf("ShutdownTimeout not applied: got %v", httpSrv.config.shutdownTimeout)
	}
	if httpSrv.config.readHeaderTimeout != 55*time.Second {
		t.Errorf("ReadHeaderTimeout not applied: got %v", httpSrv.config.readHeaderTimeout)
	}
}

func testHelperDoRequest(t *testing.T, serverAddr string, timeout time.Duration) (*http.Response, func(), error) {
	t.Helper()
	client := &http.Client{
		Timeout: timeout,
	}
	resp, err := client.Get("http://" + serverAddr + "/test")
	if err != nil {
		return nil, func() {}, err
	}
	cleanup := func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Failed to close response body: %v", err)
		}
	}
	return resp, cleanup, nil
}

func TestServerTimeoutEffects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		options     []ServerOption
		handlerFunc contracts.HTTPHandler
		expectError bool
	}{
		{
			name: "Fast handler with short timeout succeeds",
			options: []ServerOption{
				WithAddress(":0"),
				WithWriteTimeout(100 * time.Millisecond),
			},
			handlerFunc: func(ctx contracts.HTTPContext) error {
				return ctx.JSON(map[string]string{"status": "fast"})
			},
			expectError: false,
		},
		{
			name: "Slow handler with short timeout fails",
			options: []ServerOption{
				WithAddress(":0"),
				WithWriteTimeout(50 * time.Millisecond),
			},
			handlerFunc: func(ctx contracts.HTTPContext) error {
				time.Sleep(100 * time.Millisecond)
				return ctx.JSON(map[string]string{"status": "slow"})
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			logger := &mockLogger{}
			router := NewRouter(logger)
			router.GET("/test", tt.handlerFunc)
			server, _, _ := testHelperStartServer(t, router, logger, tt.options...)
			resp, cleanup, err := testHelperDoRequest(t, server.Addr(), 200*time.Millisecond)
			defer cleanup()
			if tt.expectError {
				if err == nil {
					t.Error("Expected timeout error, got success")
				}
			} else {
				if err != nil {
					t.Errorf("Expected success, got error: %v", err)
				}
				if resp != nil && resp.StatusCode != http.StatusOK {
					t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
				}
			}
		})
	}
}

func TestServerMultipleConfigChanges(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	router := NewRouter(logger)

	firstOptions := []ServerOption{
		WithAddress(":7777"),
		WithReadTimeout(10 * time.Second),
	}

	server1, err := NewServer(router, logger, firstOptions...)
	if err != nil {
		t.Fatalf("First NewServer failed: %v", err)
	}
	httpServer1 := server1.(*httpServer)

	secondOptions := []ServerOption{
		WithAddress(":8888"),
		WithReadTimeout(20 * time.Second),
		WithWriteTimeout(30 * time.Second),
	}

	server2, err := NewServer(router, logger, secondOptions...)
	if err != nil {
		t.Fatalf("Second NewServer failed: %v", err)
	}
	httpServer2 := server2.(*httpServer)

	if httpServer1.config.address != ":7777" {
		t.Error("First server address incorrect")
	}
	if httpServer1.config.readTimeout != 10*time.Second {
		t.Error("First server read timeout incorrect")
	}

	if httpServer2.config.address != ":8888" {
		t.Error("Second server address incorrect")
	}
	if httpServer2.config.readTimeout != 20*time.Second {
		t.Error("Second server read timeout incorrect")
	}
	if httpServer2.config.writeTimeout != 30*time.Second {
		t.Error("Second server write timeout incorrect")
	}
}

func TestServerAppliesReadTimeoutOnHeaders(t *testing.T) {
	t.Parallel()
	logger := &mockLogger{}
	router := NewRouter(logger)
	router.POST("/test", func(ctx contracts.HTTPContext) error {
		return ctx.String("OK")
	})
	server, err := NewServer(router, logger,
		WithAddress(":0"),
		WithReadTimeout(100*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	ctx := context.Background()
	err = server.Start(ctx)
	if err != nil {
		t.Fatalf("server.Start failed: %v", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		if err := server.Stop(stopCtx); err != nil {
			t.Logf("server.Stop failed: %v", err)
		}
	}()
	conn, err := net.Dial("tcp", server.Addr())
	if err != nil {
		t.Fatalf("net.Dial failed: %v", err)
	}
	defer func(conn net.Conn) {
		if err := conn.Close(); err != nil {
			t.Logf("CloseConn failed: %v", err)
		}
	}(conn)
	_, err = conn.Write([]byte("POST /test HTTP/1.1\r\nHost: localhost\r\n"))
	if err != nil {
		t.Fatalf("conn.Write failed: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	if err := conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
		t.Errorf("conn.SetReadDeadline failed: %v", err)
	}
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil && (err == io.EOF || err.(net.Error).Timeout()) {
		return
	}
	if err == nil && n > 0 {
		response := string(buf[:n])
		if strings.Contains(response, "408") || strings.Contains(response, "400") {
			return
		}
	}
	t.Errorf("Expected connection to be closed or 408/400 response, got: %q, err: %v", string(buf[:n]), err)
}

func TestServerStartAppliesTimeoutsToHTTPServer(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	router := NewRouter(logger)
	router.GET("/health", func(ctx contracts.HTTPContext) error {
		return ctx.JSON(map[string]string{"status": "healthy"})
	})

	expectedConfigs := map[string]time.Duration{
		"readHeaderTimeout": 3 * time.Second,
		"readTimeout":       7 * time.Second,
		"writeTimeout":      14 * time.Second,
		"idleTimeout":       21 * time.Second,
	}

	server, err := NewServer(router, logger,
		WithAddress(":0"),
		WithReadHeaderTimeout(expectedConfigs["readHeaderTimeout"]),
		WithReadTimeout(expectedConfigs["readTimeout"]),
		WithWriteTimeout(expectedConfigs["writeTimeout"]),
		WithIdleTimeout(expectedConfigs["idleTimeout"]),
	)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	ctx := context.Background()
	if err := server.Start(ctx); err != nil {
		t.Fatalf("Server start failed: %v", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		if err := server.Stop(stopCtx); err != nil {
			t.Logf("Server stop failed: %v", err)
		}
	}()

	httpSrv := server.(*httpServer)
	if httpSrv.server == nil {
		t.Fatal("HTTP server not initialized after Start")
	}

	if httpSrv.server.ReadHeaderTimeout != expectedConfigs["readHeaderTimeout"] {
		t.Errorf("ReadHeaderTimeout not applied: got %v, want %v",
			httpSrv.server.ReadHeaderTimeout, expectedConfigs["readHeaderTimeout"])
	}
	if httpSrv.server.ReadTimeout != expectedConfigs["readTimeout"] {
		t.Errorf("ReadTimeout not applied: got %v, want %v",
			httpSrv.server.ReadTimeout, expectedConfigs["readTimeout"])
	}
	if httpSrv.server.WriteTimeout != expectedConfigs["writeTimeout"] {
		t.Errorf("WriteTimeout not applied: got %v, want %v",
			httpSrv.server.WriteTimeout, expectedConfigs["writeTimeout"])
	}
	if httpSrv.server.IdleTimeout != expectedConfigs["idleTimeout"] {
		t.Errorf("IdleTimeout not applied: got %v, want %v",
			httpSrv.server.IdleTimeout, expectedConfigs["idleTimeout"])
	}
}

func TestServerHandlesRequestsAfterStart(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	router := NewRouter(logger)
	router.GET("/health", func(ctx contracts.HTTPContext) error {
		return ctx.JSON(map[string]string{"status": "healthy"})
	})

	server, err := NewServer(router, logger,
		WithAddress(":0"),
		WithReadTimeout(7*time.Second),
		WithWriteTimeout(14*time.Second),
	)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	ctx := context.Background()
	if err := server.Start(ctx); err != nil {
		t.Fatalf("Server start failed: %v", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		if err := server.Stop(stopCtx); err != nil {
			t.Logf("Server stop failed: %v", err)
		}
	}()

	resp, err := http.Get("http://" + server.Addr() + "/health")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Body.Close failed: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	if result["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %q", result["status"])
	}
}
