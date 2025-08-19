package console

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/shuldan/framework/pkg/application"
	"strings"
	"testing"
	"time"
)

type mockApplicationContext struct {
	ctx         context.Context
	container   application.Container
	appName     string
	version     string
	environment string
	startTime   time.Time
	stopTime    time.Time
	running     bool
	stopped     bool
}

func (m *mockApplicationContext) Ctx() context.Context {
	if m.ctx == nil {
		return context.Background()
	}
	return m.ctx
}

func (m *mockApplicationContext) Container() application.Container {
	return m.container
}

func (m *mockApplicationContext) AppName() string {
	return m.appName
}

func (m *mockApplicationContext) Version() string {
	return m.version
}

func (m *mockApplicationContext) Environment() string {
	return m.environment
}

func (m *mockApplicationContext) StartTime() time.Time {
	return m.startTime
}

func (m *mockApplicationContext) StopTime() time.Time {
	return m.stopTime
}

func (m *mockApplicationContext) IsRunning() bool {
	return m.running
}

func (m *mockApplicationContext) Stop() {
	m.stopped = true
	m.running = false
}

type testCommand struct {
	name        string
	description string
	group       string
	validateErr error
	executeErr  error
	executed    bool
	validated   bool
	flags       *flag.FlagSet
}

func (t *testCommand) Name() string {
	return t.name
}

func (t *testCommand) Description() string {
	return t.description
}

func (t *testCommand) Group() string {
	return t.group
}

func (t *testCommand) Configure(flags *flag.FlagSet) {
	t.flags = flags
}

func (t *testCommand) Validate(Context) error {
	t.validated = true
	return t.validateErr
}

func (t *testCommand) Execute(Context) error {
	t.executed = true
	return t.executeErr
}

func TestConsole_Register(t *testing.T) {
	registry := NewRegistry()
	c, err := New(registry)
	if err != nil {
		t.Fatalf("Failed to create console: %v", err)
	}

	cmd := &testCommand{
		name:        "test",
		description: "Test command",
		group:       "test",
	}

	err = c.Register(cmd)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	err = c.Register(cmd)
	if err == nil {
		t.Error("Expected error when registering duplicate command")
	}
}

func TestConsole_Run_NoArgs(t *testing.T) {
	registry := NewRegistry()
	_, err := New(registry)
	if err != nil {
		t.Fatalf("Failed to create console: %v", err)
	}

	appCtx := &mockApplicationContext{}

	var args []string
	ctx := newContext(appCtx, nil, nil, args)

	executor := newExecutor(newParser(registry))

	err = executor.Execute(ctx)
	if err == nil {
		t.Error("Expected error for no command specified")
	}

	if !errors.Is(err, ErrNoCommandSpecified) {
		t.Errorf("Expected ErrNoCommandSpecified, got %v", err)
	}
}

func TestConsole_Run_UnknownCommand(t *testing.T) {
	registry := NewRegistry()
	_, err := New(registry)
	if err != nil {
		t.Fatalf("Failed to create console: %v", err)
	}

	appCtx := &mockApplicationContext{}

	args := []string{"unknown"}
	ctx := newContext(appCtx, nil, nil, args)

	executor := newExecutor(newParser(registry))

	err = executor.Execute(ctx)
	if err == nil {
		t.Error("Expected error for unknown command")
	}

	if err != nil && !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("Expected unknown command error, got %v", err)
	}
}

func TestConsole_Run_ValidCommand(t *testing.T) {
	registry := NewRegistry()
	console, err := New(registry)
	if err != nil {
		t.Fatalf("Failed to create console: %v", err)
	}

	testCmd := &testCommand{
		name:        "test",
		description: "Test command",
		group:       "test",
	}

	err = console.Register(testCmd)
	if err != nil {
		t.Fatalf("Failed to register command: %v", err)
	}

	appCtx := &mockApplicationContext{}
	args := []string{"test"}

	var output bytes.Buffer
	ctx := newContext(appCtx, nil, &output, args)

	executor := newExecutor(newParser(registry))

	err = executor.Execute(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !testCmd.validated {
		t.Error("Expected command to be validated")
	}

	if !testCmd.executed {
		t.Error("Expected command to be executed")
	}
}

func TestConsole_Run_CommandValidationFailure(t *testing.T) {
	registry := NewRegistry()
	c, err := New(registry)
	if err != nil {
		t.Fatalf("Failed to create console: %v", err)
	}

	testCmd := &testCommand{
		name:        "test",
		description: "Test command",
		group:       "test",
		validateErr: fmt.Errorf("validation failed"),
	}

	err = c.Register(testCmd)
	if err != nil {
		t.Fatalf("Failed to register command: %v", err)
	}

	appCtx := &mockApplicationContext{}
	args := []string{"test"}
	ctx := newContext(appCtx, nil, nil, args)

	executor := newExecutor(newParser(registry))

	err = executor.Execute(ctx)
	if err == nil {
		t.Error("Expected validation error")
	}

	if err != nil && !strings.Contains(err.Error(), "command validation failed") {
		t.Errorf("Expected validation error, got %v", err)
	}
}

func TestConsole_Run_CommandExecutionFailure(t *testing.T) {
	registry := NewRegistry()
	c, err := New(registry)
	if err != nil {
		t.Fatalf("Failed to create console: %v", err)
	}

	testCmd := &testCommand{
		name:        "test",
		description: "Test command",
		group:       "test",
		executeErr:  fmt.Errorf("execution failed"),
	}

	err = c.Register(testCmd)
	if err != nil {
		t.Fatalf("Failed to register command: %v", err)
	}

	appCtx := &mockApplicationContext{}
	args := []string{"test"}
	ctx := newContext(appCtx, nil, nil, args)

	executor := newExecutor(newParser(registry))

	err = executor.Execute(ctx)
	if err == nil {
		t.Error("Expected execution error")
	}

	if err != nil && !strings.Contains(err.Error(), "command execution failed") {
		t.Errorf("Expected execution error, got %v", err)
	}
}
