package cli

import "github.com/shuldan/framework/pkg/contracts"

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

	parsedCtx := newContext(
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

	if err = parsed.Command.Validate(parsedCtx); err != nil {
		return ErrCommandValidation.WithDetail("command", parsed.Command.Name()).WithCause(err)
	}

	select {
	case <-parsedCtx.Ctx().Ctx().Done():
		return parsedCtx.Ctx().Ctx().Err()
	default:
	}

	if err = parsed.Command.Execute(parsedCtx); err != nil {
		return ErrCommandExecution.WithDetail("command", parsed.Command.Name()).WithCause(err)
	}

	return nil
}
