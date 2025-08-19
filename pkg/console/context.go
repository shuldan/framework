package console

import (
	"github.com/shuldan/framework/pkg/application"
	"io"
)

type cmdContext struct {
	appCtx application.Context
	input  io.Reader
	output io.Writer
	args   []string
}

func newContext(
	appCtx application.Context,
	input io.Reader,
	output io.Writer,
	args []string,
) Context {
	argsCopy := make([]string, len(args))
	copy(argsCopy, args)

	return &cmdContext{
		appCtx: appCtx,
		input:  input,
		output: output,
		args:   argsCopy,
	}
}

func (c *cmdContext) Ctx() application.Context {
	return c.appCtx
}

func (c *cmdContext) Input() io.Reader {
	return c.input
}

func (c *cmdContext) Output() io.Writer {
	return c.output
}

func (c *cmdContext) Args() []string {
	argsCopy := make([]string, len(c.args))
	copy(argsCopy, c.args)

	return argsCopy
}
