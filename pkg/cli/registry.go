package cli

import (
	"sort"
	"sync"

	"github.com/shuldan/framework/pkg/contracts"
)

type cmdRegistry struct {
	mutex    sync.RWMutex
	commands map[string]contracts.CliCommand
	groups   map[string][]string
}

func NewRegistry() contracts.CliRegistry {
	return &cmdRegistry{
		commands: make(map[string]contracts.CliCommand),
		groups:   make(map[string][]string),
	}
}

func (r *cmdRegistry) Register(command contracts.CliCommand) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if command == nil {
		return ErrCommandRegistration.WithDetail("command", "nil")
	}

	name := command.Name()
	if name == "" {
		return ErrCommandRegistration.WithDetail("command", "empty name")
	}

	if _, exists := r.commands[name]; exists {
		return ErrCommandRegistration.WithDetail("command", name).WithDetail("reason", "already registered")
	}

	r.commands[name] = command

	group := command.Group()
	if group == "" {
		group = "general"
	}
	r.groups[group] = append(r.groups[group], name)

	return nil
}

func (r *cmdRegistry) Get(name string) (contracts.CliCommand, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	command, exists := r.commands[name]
	return command, exists
}

func (r *cmdRegistry) All() map[string]contracts.CliCommand {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make(map[string]contracts.CliCommand)
	for name, command := range r.commands {
		result[name] = command
	}
	return result
}

func (r *cmdRegistry) Groups() map[string][]contracts.CliCommand {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make(map[string][]contracts.CliCommand)
	for group, names := range r.groups {
		commands := make([]contracts.CliCommand, 0, len(names))
		for _, name := range names {
			if command, exists := r.commands[name]; exists && command != nil {
				commands = append(commands, command)
			}
		}

		sort.Slice(commands, func(i, j int) bool {
			return commands[i].Name() < commands[j].Name()
		})

		result[group] = commands
	}
	return result
}
