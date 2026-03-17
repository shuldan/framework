package commandbus

import (
	"context"
	"testing"
	"time"

	"github.com/shuldan/commands"
	jsoncodec "github.com/shuldan/commands/codec/json"
	"github.com/shuldan/commands/transport/memory"
)

func newTransportAndCodec() (*memory.Transport, *jsoncodec.Codec) {
	return memory.New(), jsoncodec.New()
}

func mustClient(t *testing.T, tr commands.Transport) *commands.CommandClient {
	t.Helper()
	c, err := commands.NewCommandClient(tr, jsoncodec.New(), commands.WithTimeout(2*time.Second))
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	return c
}

func mustServer(t *testing.T, tr commands.Transport) *commands.CommandServer {
	t.Helper()
	s, err := commands.NewCommandServer(tr, jsoncodec.New())
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	return s
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestNewModule_NoOptions(t *testing.T) {
	t.Parallel()
	m := NewModule()
	if m.client != nil {
		t.Errorf("expected client to be nil, got %v", m.client)
	}
	if m.server != nil {
		t.Errorf("expected server to be nil, got %v", m.server)
	}
}

func TestNewModule_WithClient(t *testing.T) {
	t.Parallel()
	tr, _ := newTransportAndCodec()
	client := mustClient(t, tr)
	m := NewModule(WithClient(client))
	if m.client != client {
		t.Errorf("expected client to be set")
	}
	if m.server != nil {
		t.Errorf("expected server to be nil")
	}
}

func TestNewModule_WithServer(t *testing.T) {
	t.Parallel()
	tr, _ := newTransportAndCodec()
	server := mustServer(t, tr)
	m := NewModule(WithServer(server))
	if m.server != server {
		t.Errorf("expected server to be set")
	}
	if m.client != nil {
		t.Errorf("expected client to be nil")
	}
}

func TestNewModule_WithBoth(t *testing.T) {
	t.Parallel()
	tr := memory.New()
	client := mustClient(t, tr)
	server := mustServer(t, tr)
	m := NewModule(WithClient(client), WithServer(server))
	if m.client != client {
		t.Errorf("expected client to be set")
	}
	if m.server != server {
		t.Errorf("expected server to be set")
	}
}

func TestModule_Name(t *testing.T) {
	t.Parallel()
	m := NewModule()
	if got := m.Name(); got != "commandbus" {
		t.Errorf("expected 'commandbus', got %q", got)
	}
}

func TestModule_Init(t *testing.T) {
	t.Parallel()
	m := NewModule()
	if err := m.Init(context.Background()); err != nil {
		t.Errorf("expected nil error from Init, got %v", err)
	}
}

func TestModule_Init_CancelledContext(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m := NewModule()
	if err := m.Init(ctx); err != nil {
		t.Errorf("Init should not fail even with cancelled context, got %v", err)
	}
}

func TestModule_Client_Accessor(t *testing.T) {
	t.Parallel()
	tr := memory.New()
	client := mustClient(t, tr)
	m := NewModule(WithClient(client))
	if m.Client() != client {
		t.Errorf("Client() did not return expected client")
	}
}

func TestModule_Server_Accessor(t *testing.T) {
	t.Parallel()
	tr := memory.New()
	server := mustServer(t, tr)
	m := NewModule(WithServer(server))
	if m.Server() != server {
		t.Errorf("Server() did not return expected server")
	}
}

func TestModule_Client_Nil(t *testing.T) {
	t.Parallel()
	m := NewModule()
	if m.Client() != nil {
		t.Errorf("expected nil client")
	}
}

func TestModule_Server_Nil(t *testing.T) {
	t.Parallel()
	m := NewModule()
	if m.Server() != nil {
		t.Errorf("expected nil server")
	}
}

func TestModule_Start_NilBoth(t *testing.T) {
	t.Parallel()
	m := NewModule()
	if err := m.Start(context.Background()); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestModule_Start_ServerOnly(t *testing.T) {
	t.Parallel()
	tr := memory.New()
	server := mustServer(t, tr)
	m := NewModule(WithServer(server))
	ctx := context.Background()
	if err := m.Start(ctx); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	t.Cleanup(func() { _ = m.Stop(ctx) })
}

func TestModule_Start_ClientOnly(t *testing.T) {
	t.Parallel()
	tr := memory.New()
	client := mustClient(t, tr)
	m := NewModule(WithClient(client))
	ctx := context.Background()
	if err := m.Start(ctx); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	t.Cleanup(func() { _ = m.Stop(ctx) })
}

func TestModule_Start_Both(t *testing.T) {
	t.Parallel()
	tr := memory.New()
	client := mustClient(t, tr)
	server := mustServer(t, tr)
	m := NewModule(WithClient(client), WithServer(server))
	ctx := context.Background()
	if err := m.Start(ctx); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	t.Cleanup(func() { _ = m.Stop(ctx) })
}

func TestModule_Start_ServerOpenError(t *testing.T) {
	t.Parallel()
	tr := memory.New()
	server := mustServer(t, tr)
	ctx := context.Background()
	_ = server.Open(ctx)
	m := NewModule(WithServer(server))
	err := m.Start(ctx)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !contains(err.Error(), "commandbus: open server") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestModule_Start_ClientOpenError(t *testing.T) {
	t.Parallel()
	tr := memory.New()
	client := mustClient(t, tr)
	ctx := context.Background()
	_ = client.Open(ctx)
	m := NewModule(WithClient(client))
	err := m.Start(ctx)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !contains(err.Error(), "commandbus: open client") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestModule_Start_ServerErrorPreventsClientOpen(t *testing.T) {
	t.Parallel()
	tr := memory.New()
	server := mustServer(t, tr)
	_ = server.Open(context.Background())
	client := mustClient(t, tr)
	m := NewModule(WithServer(server), WithClient(client))
	err := m.Start(context.Background())
	if err == nil {
		t.Fatalf("expected error")
	}
	if !contains(err.Error(), "open server") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestModule_Stop_NilBoth(t *testing.T) {
	t.Parallel()
	m := NewModule()
	if err := m.Stop(context.Background()); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestModule_Stop_ClientOnly_Success(t *testing.T) {
	t.Parallel()
	tr := memory.New()
	client := mustClient(t, tr)
	m := NewModule(WithClient(client))
	ctx := context.Background()
	_ = m.Start(ctx)
	if err := m.Stop(ctx); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestModule_Stop_ServerOnly_Success(t *testing.T) {
	t.Parallel()
	tr := memory.New()
	server := mustServer(t, tr)
	m := NewModule(WithServer(server))
	ctx := context.Background()
	_ = m.Start(ctx)
	if err := m.Stop(ctx); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestModule_Stop_Both_Success(t *testing.T) {
	t.Parallel()
	tr := memory.New()
	client := mustClient(t, tr)
	server := mustServer(t, tr)
	m := NewModule(WithClient(client), WithServer(server))
	ctx := context.Background()
	_ = m.Start(ctx)
	if err := m.Stop(ctx); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestModule_Stop_ClientCloseError(t *testing.T) {
	t.Parallel()
	client := mustClient(t, memory.New())
	m := NewModule(WithClient(client))
	err := m.Stop(context.Background())
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !contains(err.Error(), "commandbus: close client") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestModule_Stop_ServerCloseError(t *testing.T) {
	t.Parallel()
	server := mustServer(t, memory.New())
	m := NewModule(WithServer(server))
	err := m.Stop(context.Background())
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !contains(err.Error(), "commandbus: close server") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestModule_Stop_BothErrors_ClientFirst(t *testing.T) {
	t.Parallel()
	client := mustClient(t, memory.New())
	server := mustServer(t, memory.New())
	m := NewModule(WithClient(client), WithServer(server))
	err := m.Stop(context.Background())
	if err == nil {
		t.Fatalf("expected error")
	}
	if !contains(err.Error(), "commandbus: close client") {
		t.Errorf("expected client error first, got: %v", err)
	}
}

func TestModule_Stop_ClientOk_ServerError(t *testing.T) {
	t.Parallel()
	clientTr := memory.New()
	serverTr := memory.New()
	client := mustClient(t, clientTr)
	server := mustServer(t, serverTr)
	m := NewModule(WithClient(client), WithServer(server))
	ctx := context.Background()
	if err := client.Open(ctx); err != nil {
		t.Fatalf("client open: %v", err)
	}
	if err := m.Stop(ctx); err == nil {
		t.Fatalf("expected error from server close, got nil")
	} else if !contains(err.Error(), "commandbus: close server") {
		t.Errorf("expected server error, got: %v", err)
	}
}

func TestModule_StartStop_FullLifecycle(t *testing.T) {
	t.Parallel()
	tr := memory.New()
	client := mustClient(t, tr)
	server := mustServer(t, tr)
	m := NewModule(WithClient(client), WithServer(server))
	ctx := context.Background()
	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if err := m.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}
