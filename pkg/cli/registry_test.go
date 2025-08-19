package cli

import (
	"github.com/shuldan/framework/pkg/contracts"
	"sync"
	"testing"
)

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	cmd := &testCommand{
		name:        "test",
		description: "Test command",
		group:       "test",
	}

	err := registry.Register(cmd)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	err = registry.Register(nil)
	if err == nil {
		t.Error("Expected error for nil command")
	}

	emptyCmd := &testCommand{
		name: "",
	}
	err = registry.Register(emptyCmd)
	if err == nil {
		t.Error("Expected error for empty command name")
	}

	duplicateCmd := &testCommand{
		name: "test",
	}
	err = registry.Register(duplicateCmd)
	if err == nil {
		t.Error("Expected error for duplicate command")
	}
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()

	cmd := &testCommand{
		name:        "test",
		description: "Test command",
		group:       "test",
	}

	err := registry.Register(cmd)
	if err != nil {
		t.Fatalf("Failed to register command: %v", err)
	}

	retrievedCmd, exists := registry.Get("test")
	if !exists {
		t.Error("Expected command to exist")
	}
	if retrievedCmd != cmd {
		t.Error("Expected same command instance")
	}

	_, exists = registry.Get("nonexistent")
	if exists {
		t.Error("Expected command to not exist")
	}
}

func TestRegistry_Groups(t *testing.T) {
	registry := NewRegistry()

	cmd1 := &testCommand{
		name:        "cmd1",
		description: "CliCommand 1",
		group:       "group1",
	}

	cmd2 := &testCommand{
		name:        "cmd2",
		description: "CliCommand 2",
		group:       "group1",
	}

	cmd3 := &testCommand{
		name:        "cmd3",
		description: "CliCommand 3",
		group:       "group2",
	}

	var errs []error

	if err := registry.Register(cmd1); err != nil {
		errs = append(errs, err)
	}
	if err := registry.Register(cmd2); err != nil {
		errs = append(errs, err)
	}
	if err := registry.Register(cmd3); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		t.Error("Failed to register commands")
	}

	groups := registry.Groups()

	if len(groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(groups))
	}

	if len(groups["group1"]) != 2 {
		t.Errorf("Expected 2 commands in group1, got %d", len(groups["group1"]))
	}

	if len(groups["group2"]) != 1 {
		t.Errorf("Expected 1 command in group2, got %d", len(groups["group2"]))
	}

	if groups["group1"][0].Name() != "cmd1" {
		t.Error("Expected cmd1 to be first in group1")
	}
	if groups["group1"][1].Name() != "cmd2" {
		t.Error("Expected cmd2 to be second in group1")
	}
}

func TestRegistry_All(t *testing.T) {
	registry := NewRegistry().(*cmdRegistry)

	cmd1 := &testCommand{
		name:        "cmd1",
		description: "CliCommand 1",
		group:       "test",
	}

	cmd2 := &testCommand{
		name:        "cmd2",
		description: "CliCommand 2",
		group:       "test",
	}

	err := registry.Register(cmd1)
	if err != nil {
		t.Fatalf("Failed to register cmd1: %v", err)
	}

	err = registry.Register(cmd2)
	if err != nil {
		t.Fatalf("Failed to register cmd2: %v", err)
	}

	all := registry.All()
	if len(all) != 2 {
		t.Errorf("Expected 2 commands, got %d", len(all))
	}

	if all["cmd1"] != cmd1 {
		t.Error("Expected cmd1 to be in all map")
	}

	if all["cmd2"] != cmd2 {
		t.Error("Expected cmd2 to be in all map")
	}
}

func TestRegistry_Groups_EmptyGroup(t *testing.T) {
	registry := NewRegistry()

	cmd := &testCommand{
		name:        "cmd",
		description: "CliCommand with empty group",
		group:       "",
	}

	err := registry.Register(cmd)
	if err != nil {
		t.Fatalf("Failed to register command: %v", err)
	}

	groups := registry.Groups()
	if len(groups["general"]) != 1 {
		t.Error("Expected command to be in 'general' group")
	}
}

func TestRegistry_Groups_NilCommand(t *testing.T) {
	r := &cmdRegistry{
		mutex:    sync.RWMutex{},
		commands: make(map[string]contracts.CliCommand),
		groups:   make(map[string][]string),
	}

	r.mutex.Lock()
	r.commands["nilcmd"] = nil
	r.groups["test"] = []string{"nilcmd"}
	r.mutex.Unlock()

	groups := r.Groups()
	if len(groups["test"]) != 0 {
		t.Error("Expected nil command to be filtered out")
	}
}
