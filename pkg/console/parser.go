package console

import (
	"flag"
	"io"
)

type parsedCommand struct {
	Name    string
	Args    []string
	Flags   *flag.FlagSet
	Command Command
}

type cmdParser struct {
	registry Registry
}

func newParser(registry Registry) *cmdParser {
	return &cmdParser{
		registry: registry,
	}
}

func (p *cmdParser) Parse(args []string, output io.Writer) (*parsedCommand, error) {
	if len(args) == 0 {
		return nil, ErrNoCommandSpecified
	}

	commandName := args[0]
	commandArgs := args[1:]

	command, exists := p.registry.Get(commandName)
	if !exists {
		return nil, ErrUnknownCommand.WithDetail("command", commandName)
	}

	flagSet := flag.NewFlagSet(commandName, flag.ContinueOnError)
	flagSet.SetOutput(output)

	command.Configure(flagSet)

	if err := flagSet.Parse(commandArgs); err != nil {
		return nil, ErrFlagParse.WithDetail("command", commandName).WithCause(err)
	}

	return &parsedCommand{
		Name:    commandName,
		Args:    flagSet.Args(),
		Flags:   flagSet,
		Command: command,
	}, nil
}
