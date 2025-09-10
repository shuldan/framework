package contracts

import (
	"flag"
	"io"
)

const (
	SystemCliGroup   = "system"
	DatabaseCliGroup = "database"
	HttpCliGroup     = "http"
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
	Run(ctx CliContext) error
}

type CliCommandProvider interface {
	CliCommands(ctx AppContext) ([]CliCommand, error)
}
