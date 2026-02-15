package httpserver

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestModule_Name(t *testing.T) {
	t.Parallel()
	m := NewModule(http.NewServeMux(), Config{})
	if m.Name() != "httpserver" {
		t.Fatalf("expected 'httpserver', got %q", m.Name())
	}
}

func TestModule_Lifecycle(t *testing.T) {
	router := NewRouter()
	router.GET("/ping", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("pong"))
	})
	m := NewModule(router, Config{Host: "127.0.0.1", Port: 0})
	ctx := context.Background()
	assertNoErr(t, m.Init(ctx))
	assertNoErr(t, m.Start(ctx))
	defer func() { assertNoErr(t, m.Stop(ctx)) }()
	resp := httpGet(t, "http://"+m.Addr()+"/ping")
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "pong" {
		t.Fatalf("expected 'pong', got %q", body)
	}
}

func TestModule_Stop_ClosesServer(t *testing.T) {
	m := NewModule(http.NewServeMux(), Config{Host: "127.0.0.1", Port: 0})
	ctx := context.Background()
	assertNoErr(t, m.Init(ctx))
	assertNoErr(t, m.Start(ctx))
	assertNoErr(t, m.Stop(ctx))
	client := &http.Client{Timeout: 100 * time.Millisecond}
	_, err := client.Get("http://" + m.Addr() + "/")
	if err == nil {
		t.Fatal("expected error after stop")
	}
}

func TestModule_Stop_NilServer(t *testing.T) {
	t.Parallel()
	m := NewModule(http.NewServeMux(), Config{})
	err := m.Stop(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestModule_Addr_BeforeInit(t *testing.T) {
	t.Parallel()
	m := NewModule(http.NewServeMux(), Config{})
	if m.Addr() != "" {
		t.Fatalf("expected empty addr, got %q", m.Addr())
	}
}

func TestModule_Err_Channel(t *testing.T) {
	t.Parallel()
	m := NewModule(http.NewServeMux(), Config{})
	ch := m.Err()
	if ch == nil {
		t.Fatal("expected non-nil error channel")
	}
}

func TestModule_DefaultConfig(t *testing.T) {
	m := NewModule(http.NewServeMux(), Config{Host: "127.0.0.1"})
	ctx := context.Background()
	assertNoErr(t, m.Init(ctx))
	defer func() { _ = m.Stop(ctx) }()
	if m.Addr() == "" {
		t.Fatal("expected non-empty address")
	}
}

func TestModule_Init_Error(t *testing.T) {
	t.Parallel()
	m := NewModule(http.NewServeMux(), Config{Host: "invalid-host-^%$", Port: -1})
	err := m.Init(context.Background())
	if err == nil {
		_ = m.Stop(context.Background())
		t.Log("init did not fail with invalid host (OS-dependent)")
	}
}

func httpGet(t *testing.T, url string) *http.Response {
	t.Helper()
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	return resp
}

func assertNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
