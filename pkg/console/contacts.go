package console

import (
	"flag"
	"github.com/shuldan/framework/pkg/application"
	"io"
)

type Context interface {
	Ctx() application.Context
	Input() io.Reader
	Output() io.Writer
	Args() []string
}

type Command interface {
	Name() string
	Description() string
	Group() string
	Configure(flags *flag.FlagSet)
	Validate(ctx Context) error
	Execute(ctx Context) error
}

type Registry interface {
	Register(command Command) error
	Get(name string) (Command, bool)
	Groups() map[string][]Command
}

type Console interface {
	Register(cmd Command) error
	Run(appCtx application.Context) error
}
