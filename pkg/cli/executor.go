package cli

import (
	"fmt"

	"github.com/shuldan/framework/pkg/contracts"
)

type cmdExecutor struct {
	parser *cmdParser
}

func newExecutor(parser *cmdParser) *cmdExecutor {
	return &cmdExecutor{
		parser: parser,
	}
}

func (e *cmdExecutor) Execute(commandCtx contracts.CliContext) error {
	select {
	case <-commandCtx.Ctx().Ctx().Done():
		return commandCtx.Ctx().Ctx().Err()
	default:
	}

	if len(commandCtx.Args()) == 0 {
		return ErrNoCommandSpecified
	}

	parsed, err := e.parser.Parse(commandCtx.Args(), commandCtx.Output())
	if err != nil {
		return err
	}

	select {
	case <-commandCtx.Ctx().Ctx().Done():
		return commandCtx.Ctx().Ctx().Err()
	default:
	}

	parsedCtx := NewContext(
		commandCtx.Ctx(),
		commandCtx.Input(),
		commandCtx.Output(),
		parsed.Args,
	)

	select {
	case <-parsedCtx.Ctx().Ctx().Done():
		return parsedCtx.Ctx().Ctx().Err()
	default:
	}

	if err = e.withRecovery(func() error {
		return parsed.Command.Validate(parsedCtx)
	}); err != nil {
		return ErrCommandValidation.WithDetail("command", parsed.Command.Name()).WithCause(err)
	}

	select {
	case <-parsedCtx.Ctx().Ctx().Done():
		return parsedCtx.Ctx().Ctx().Err()
	default:
	}

	if err = e.withRecovery(func() error {
		return parsed.Command.Execute(parsedCtx)
	}); err != nil {
		return ErrCommandExecution.WithDetail("command", parsed.Command.Name()).WithCause(err)
	}

	return nil
}

func (e *cmdExecutor) withRecovery(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during command execution: %v", r)
		}
	}()
	return fn()
}
