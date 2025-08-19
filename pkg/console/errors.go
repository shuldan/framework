package console

import "github.com/shuldan/framework/pkg/errors"

var newConsoleCode = errors.WithPrefix("CONSOLE")

var (
	ErrNoCommandSpecified     = newConsoleCode().New("no command specified")
	ErrUnknownCommand         = newConsoleCode().New("unknown command {{.command}}")
	ErrCommandValidation      = newConsoleCode().New("command validation failed for {{.command}}")
	ErrCommandExecution       = newConsoleCode().New("command execution failed for {{.command}}")
	ErrCommandRegistration    = newConsoleCode().New("command registration failed for {{.command}}")
	ErrHelpCommandNotFound    = newConsoleCode().New("help command not found for {{.command}}")
	ErrInvalidConsoleInstance = newConsoleCode().New("console not implemented Console interface")
	ErrFlagParse              = newConsoleCode().New("flag parsing failed for command {{.command}}")
)
