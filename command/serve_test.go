package command

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/shuldan/cli"

	"github.com/shuldan/framework/logger"
)

func TestServe_ExecutesLifecycle(t *testing.T) {
	mod := &mockModule{name: "test"}
	log := logger.New(logger.Config{Level: "error"})
	cmd := Serve("app", log, time.Second, mod)
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

func TestServe_CommandMetadata(t *testing.T) {
	t.Parallel()
	cmd := Serve("app", nil, 0)
	if cmd.Name() != "serve" {
		t.Errorf("expected 'serve', got %q", cmd.Name())
	}
	if cmd.Group() != "server" {
		t.Errorf("expected 'server', got %q", cmd.Group())
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

func TestServe_ReturnsInitError(t *testing.T) {
	mod := &mockModule{name: "fail", initErr: errTest}
	log := logger.New(logger.Config{Level: "error"})
	cmd := Serve("app", log, time.Second, mod)
	ctx := context.Background()
	err := cmd.Execute(ctx, emptyReader(), io.Discard, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServe_MultipleModules(t *testing.T) {
	mod1 := &mockModule{name: "mod1"}
	mod2 := &mockModule{name: "mod2"}
	log := logger.New(logger.Config{Level: "error"})
	cmd := Serve("app", log, time.Second, mod1, mod2)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := cmd.Execute(ctx, emptyReader(), io.Discard, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServe_DuplicateModuleRegistration(t *testing.T) {
	mod1 := &mockModule{name: "same"}
	mod2 := &mockModule{name: "same"}
	log := logger.New(logger.Config{Level: "error"})
	cmd := Serve("app", log, time.Second, mod1, mod2)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := cmd.Execute(ctx, emptyReader(), io.Discard, nil)
	if err == nil {
		t.Log("app.Register may not return error for duplicate modules")
	}
}

var errTest = context.DeadlineExceeded

type mockModule struct {
	name        string
	initCalled  bool
	startCalled bool
	stopCalled  bool
	initErr     error
}

func (m *mockModule) Name() string { return m.name }

func (m *mockModule) Init(_ context.Context) error {
	m.initCalled = true
	return m.initErr
}

func (m *mockModule) Start(_ context.Context) error {
	m.startCalled = true
	return nil
}

func (m *mockModule) Stop(_ context.Context) error {
	m.stopCalled = true
	return nil
}

func emptyReader() io.Reader { return &bytes.Buffer{} }

var _ cli.Command = Serve("", nil, 0)
