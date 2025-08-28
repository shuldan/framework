package cli

import (
	"bytes"
	"errors"
	"flag"
	"testing"

	"github.com/shuldan/framework/pkg/contracts"
)

type flagTestCommand struct {
	name        string
	description string
	group       string
	value       string
	validateErr error
	executeErr  error
	executed    bool
	validated   bool
	flags       *flag.FlagSet
}

func (f *flagTestCommand) Name() string {
	return f.name
}

func (f *flagTestCommand) Description() string {
	return f.description
}

func (f *flagTestCommand) Group() string {
	return f.group
}

func (f *flagTestCommand) Configure(flags *flag.FlagSet) {
	f.flags = flags
	flags.StringVar(&f.value, "value", "", "Test value")
}

func (f *flagTestCommand) Validate(contracts.CliContext) error {
	f.validated = true
	return f.validateErr
}

func (f *flagTestCommand) Execute(contracts.CliContext) error {
	f.executed = true
	return f.executeErr
}

func TestParser_Parse(t *testing.T) {
	registry := NewRegistry()
	parser := newParser(registry)

	_, err := parser.Parse([]string{}, nil)
	if err == nil {
		t.Error("Expected error for no args")
	}

	_, err = parser.Parse([]string{"unknown"}, nil)
	if err == nil {
		t.Error("Expected error for unknown command")
	}

	cmd := &testCommand{
		name:        "test",
		description: "Test command",
		group:       "test",
	}

	err = registry.Register(cmd)
	if err != nil {
		t.Fatalf("Failed to register command: %v", err)
	}

	var output bytes.Buffer
	parsed, err := parser.Parse([]string{"test", "arg1", "arg2"}, &output)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if parsed.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", parsed.Name)
	}

	if len(parsed.Args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(parsed.Args))
	}

	if parsed.Command != cmd {
		t.Error("Expected same command instance")
	}
}

func TestParser_ParseWithFlags(t *testing.T) {
	registry := NewRegistry()
	parser := newParser(registry)

	cmd := &flagTestCommand{
		name:        "test",
		description: "Test command with flags",
		group:       "test",
	}

	err := registry.Register(cmd)
	if err != nil {
		t.Fatalf("Failed to register command: %v", err)
	}

	var output bytes.Buffer
	parsed, err := parser.Parse([]string{"test", "-value=testvalue", "arg1"}, &output)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if parsed.Flags == nil {
		t.Error("Expected flags to be set")
	}

	if len(parsed.Args) != 1 {
		t.Errorf("Expected 1 remaining arg, got %d", len(parsed.Args))
	}

	if parsed.Args[0] != "arg1" {
		t.Errorf("Expected arg 'arg1', got '%s'", parsed.Args[0])
	}

	value := parsed.Flags.Lookup("value")
	if value == nil {
		t.Error("Expected 'value' flag to be set")
	} else if value.Value.String() != "testvalue" {
		t.Errorf("Expected flag value 'testvalue', got '%s'", value.Value.String())
	}
}

func TestParser_ParseFlagError(t *testing.T) {
	registry := NewRegistry()
	parser := newParser(registry)

	cmd := &flagTestCommand{
		name:        "test",
		description: "Test command with flags",
		group:       "test",
	}

	err := registry.Register(cmd)
	if err != nil {
		t.Fatalf("Failed to register command: %v", err)
	}

	var output bytes.Buffer
	_, err = parser.Parse([]string{"test", "-invalid=value"}, &output)
	if err == nil {
		t.Error("Expected error for invalid flag")
	}

	if !errors.Is(err, ErrFlagParse) {
		t.Errorf("Expected ErrFlagParse, got %v", err)
	}
}

func TestParser_EmptyArgs(t *testing.T) {
	registry := NewRegistry()
	parser := newParser(registry)

	var output bytes.Buffer
	_, err := parser.Parse([]string{}, &output)
	if err == nil {
		t.Error("Expected error for empty args")
	}

	if !errors.Is(err, ErrNoCommandSpecified) {
		t.Errorf("Expected ErrNoCommandSpecified, got %v", err)
	}
}
