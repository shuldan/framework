package contracts

import (
	"flag"
	"io"
)

type CliContext interface {
	Ctx() AppContext
	Input() io.Reader
	Output() io.Writer
	Args() []string
}

type CliCommand interface {
	Name() string
	Description() string
	Group() string
	Configure(flags *flag.FlagSet)
	Validate(ctx CliContext) error
	Execute(ctx CliContext) error
}

type CliRegistry interface {
	Register(command CliCommand) error
	Get(name string) (CliCommand, bool)
	Groups() map[string][]CliCommand
}

type Cli interface {
	Register(cmd CliCommand) error
	Run(appCtx AppContext) error
}
