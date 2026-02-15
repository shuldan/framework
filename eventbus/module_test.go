package eventbus

import (
	"context"
	"testing"
	"time"

	"github.com/shuldan/events"
)

func TestModule_Name(t *testing.T) {
	t.Parallel()
	m := NewModule(Config{})
	if m.Name() != "eventbus" {
		t.Fatalf("expected 'eventbus', got %q", m.Name())
	}
}

func TestModule_Dispatcher_NotNil(t *testing.T) {
	t.Parallel()
	m := NewModule(Config{})
	if m.Dispatcher() == nil {
		t.Fatal("expected non-nil dispatcher")
	}
}

func TestModule_Lifecycle(t *testing.T) {
	t.Parallel()
	m := NewModule(Config{})
	ctx := context.Background()
	if err := m.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := m.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

func TestModule_SyncConfig(t *testing.T) {
	m := NewModule(Config{Async: false})
	handled := false
	events.SubscribeFunc(m.Dispatcher(), func(_ context.Context, _ *testEvent) error {
		handled = true
		return nil
	})
	ctx := context.Background()
	err := m.Dispatcher().Publish(ctx, &testEvent{
		BaseEvent: events.NewBaseEvent("test", "agg-1"),
	})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if !handled {
		t.Fatal("event not handled in sync mode")
	}
	_ = m.Stop(ctx)
}

func TestModule_AsyncConfig(t *testing.T) {
	m := NewModule(Config{Async: true, Workers: 2, BufferSize: 10})
	handled := make(chan struct{}, 1)
	events.SubscribeFunc(m.Dispatcher(), func(_ context.Context, _ *testEvent) error {
		handled <- struct{}{}
		return nil
	})
	ctx := context.Background()
	err := m.Dispatcher().Publish(ctx, &testEvent{
		BaseEvent: events.NewBaseEvent("test", "agg-1"),
	})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	select {
	case <-handled:
	case <-time.After(time.Second):
		t.Fatal("event not handled in async mode")
	}
	_ = m.Stop(ctx)
}

type testEvent struct {
	events.BaseEvent
	Value string `json:"value"`
}
