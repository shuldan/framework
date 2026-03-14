package commandbus

import (
	"context"
	"testing"
	"time"
)

func TestNewModule_CreatesDispatcher(t *testing.T) {
	t.Parallel()
	m := NewModule(Config{Async: false})
	if m.Dispatcher() == nil {
		t.Fatal("expected non-nil dispatcher")
	}
}

func TestModule_Name(t *testing.T) {
	t.Parallel()
	m := NewModule(Config{})
	if m.Name() != "commandbus" {
		t.Errorf("expected %q, got %q", "commandbus", m.Name())
	}
}

func TestModule_Init(t *testing.T) {
	t.Parallel()
	m := NewModule(Config{})
	if err := m.Init(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestModule_Start(t *testing.T) {
	t.Parallel()
	m := NewModule(Config{})
	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestModule_Stop(t *testing.T) {
	t.Parallel()
	m := NewModule(Config{Async: false})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := m.Stop(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestModule_AsyncStop(t *testing.T) {
	t.Parallel()
	m := NewModule(Config{Async: true, Workers: 2, BufferSize: 10})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := m.Stop(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
