package cli

import (
	"bytes"
	"context"
	"flag"
	"strings"
	"testing"
	"time"

	"github.com/shuldan/framework/pkg/contracts"
)

type helpAppContext struct {
	stopped bool
}

func (h *helpAppContext) Ctx() context.Context {
	return context.Background()
}

func (h *helpAppContext) Container() contracts.DIContainer {
	return nil
}

func (h *helpAppContext) AppName() string {
	return testAppName
}

func (h *helpAppContext) Version() string {
	return testVersion
}

func (h *helpAppContext) Environment() string {
	return testEnv
}

func (h *helpAppContext) StartTime() time.Time {
	return time.Now()
}

func (h *helpAppContext) StopTime() time.Time {
	return time.Time{}
}

func (h *helpAppContext) IsRunning() bool {
	return !h.stopped
}

func (h *helpAppContext) Stop() {
	h.stopped = true
}

func TestHelpCommand(t *testing.T) {
	registry := NewRegistry()
	helpCmd := NewHelpCommand(registry)

	if helpCmd.Name() != "help" {
		t.Errorf("Expected name 'help', got '%s'", helpCmd.Name())
	}

	if helpCmd.Description() != "Display help for commands" {
		t.Errorf("Expected Description 'Display help for commands', got '%s'", helpCmd.Description())
	}

	if helpCmd.Group() != "system" {
		t.Errorf("Expected group 'system', got '%s'", helpCmd.Group())
	}

	flags := flag.NewFlagSet("help", flag.ContinueOnError)
	helpCmd.Configure(flags)

	err := helpCmd.Validate(nil)
	if err != nil {
		t.Errorf("Expected no error from Validate, got %v", err)
	}
}

func TestHelpCommand_Execute_ShowGeneralHelp(t *testing.T) {
	registry := NewRegistry()
	helpCmd := NewHelpCommand(registry)

	testCmd := &testCommand{
		name:        "test",
		description: "Test command",
		group:       "test",
	}

	if err := registry.Register(testCmd); err != nil {
		t.Errorf("Expected no error from Register, got %v", err)
	}

	appCtx := &helpAppContext{}
	var output bytes.Buffer
	ctx := newContext(appCtx, nil, &output, []string{})

	if err := helpCmd.Execute(ctx); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "test") {
		t.Error("Expected help output to contain 'test' command")
	}

	if !strings.Contains(outputStr, "Test command") {
		t.Error("Expected help output to contain command Description")
	}

	if !appCtx.stopped {
		t.Error("Expected application context to be stopped")
	}
}

func TestHelpCommand_Execute_ShowCommandHelp(t *testing.T) {
	registry := NewRegistry()
	helpCmd := NewHelpCommand(registry).(*HelpCommand)
	helpCmd.command = "test"

	testCmd := &testCommand{
		name:        "test",
		description: "Test command",
		group:       "test",
	}
	if err := registry.Register(testCmd); err != nil {
		t.Errorf("Expected no error from Register, got %v", err)
	}

	appCtx := &helpAppContext{}
	var output bytes.Buffer
	ctx := newContext(appCtx, nil, &output, []string{})

	err := helpCmd.Execute(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "test - Test command") {
		t.Error("Expected help output to contain command name and Description")
	}

	if !appCtx.stopped {
		t.Error("Expected application context to be stopped")
	}
}

func TestHelpCommand_Execute_ShowCommandHelp_NotFound(t *testing.T) {
	registry := NewRegistry()
	helpCmd := NewHelpCommand(registry).(*HelpCommand)
	helpCmd.command = "nonexistent"

	appCtx := &helpAppContext{}
	var output bytes.Buffer
	ctx := newContext(appCtx, nil, &output, []string{})

	err := helpCmd.Execute(ctx)
	if err == nil {
		t.Error("Expected error for nonexistent command")
	}

	if err != nil && !strings.Contains(err.Error(), "help command not found") {
		t.Errorf("Expected help command not found error, got %v", err)
	}
}
