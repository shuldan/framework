package console

import (
	"context"
	"testing"
)

func TestExecutor_Execute(t *testing.T) {
	registry := NewRegistry()
	parser := newParser(registry)
	executor := newExecutor(parser)

	appCtx := &simpleAppContext{}
	ctx := newContext(appCtx, nil, nil, []string{})
	err := executor.Execute(ctx)
	if err == nil {
		t.Error("Expected error for no args")
	}
}

func TestExecutor_Execute_CancelledContext(t *testing.T) {
	registry := NewRegistry()
	parser := newParser(registry)
	executor := newExecutor(parser)

	cmd := &testCommand{
		name:        "test",
		description: "Test command",
		group:       "test",
	}

	err := registry.Register(cmd)
	if err != nil {
		t.Fatalf("Failed to register command: %v", err)
	}

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	cancelledAppCtx := &simpleAppContext{ctx: cancelledCtx}
	ctx := newContext(cancelledAppCtx, nil, nil, []string{"test"})
	err = executor.Execute(ctx)
	if err == nil {
		t.Error("Expected error for cancelled context")
	}
}
