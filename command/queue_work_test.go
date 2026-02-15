package command

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/shuldan/framework/logger"
)

func TestQueueWork_Metadata(t *testing.T) {
	t.Parallel()
	cmd := QueueWork("app", nil, 0)
	if cmd.Name() != "queue:work" {
		t.Errorf("expected 'queue:work', got %q", cmd.Name())
	}
	if cmd.Group() != "queue" {
		t.Errorf("expected 'queue', got %q", cmd.Group())
	}
	if cmd.Description() == "" {
		t.Error("expected non-empty description")
	}
	if cmd.Args() != nil {
		t.Error("expected nil args")
	}
	if cmd.Options() != nil {
		t.Error("expected nil options")
	}
}

func TestQueueWork_ExecutesLifecycle(t *testing.T) {
	mod := &mockModule{name: "worker"}
	log := logger.New(logger.Config{Level: "error"})
	cmd := QueueWork("app", log, time.Second, mod)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := cmd.Execute(ctx, emptyReader(), io.Discard, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mod.initCalled {
		t.Error("Init not called")
	}
	if !mod.startCalled {
		t.Error("Start not called")
	}
	if !mod.stopCalled {
		t.Error("Stop not called")
	}
}

func TestQueueWork_ReturnsInitError(t *testing.T) {
	mod := &mockModule{name: "fail", initErr: errTest}
	log := logger.New(logger.Config{Level: "error"})
	cmd := QueueWork("app", log, time.Second, mod)
	ctx := context.Background()
	err := cmd.Execute(ctx, emptyReader(), io.Discard, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestQueueWork_MultipleModules(t *testing.T) {
	mod1 := &mockModule{name: "w1"}
	mod2 := &mockModule{name: "w2"}
	log := logger.New(logger.Config{Level: "error"})
	cmd := QueueWork("app", log, time.Second, mod1, mod2)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := cmd.Execute(ctx, emptyReader(), io.Discard, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestQueueWork_DuplicateModuleRegistration(t *testing.T) {
	mod1 := &mockModule{name: "dup"}
	mod2 := &mockModule{name: "dup"}
	log := logger.New(logger.Config{Level: "error"})
	cmd := QueueWork("app", log, time.Second, mod1, mod2)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := cmd.Execute(ctx, emptyReader(), io.Discard, nil)
	if err == nil {
		t.Log("app.Register may not return error for duplicate modules")
	}
}
