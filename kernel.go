package framework

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/shuldan/cli"
	"github.com/shuldan/config"

	"github.com/shuldan/framework/logger"
)

type Kernel struct {
	cfg      *config.Config
	log      *logger.Logger
	console  *cli.Console
	cleanups []func()
}

func NewKernel(opts ...KernelOption) (*Kernel, error) {
	o := defaultKernelOptions()
	for _, opt := range opts {
		opt(o)
	}

	cfg, err := buildConfig(o)
	if err != nil {
		return nil, fmt.Errorf("framework: load config: %w", err)
	}

	log := buildLogger(cfg, o)
	console := buildConsole(cfg)

	return &Kernel{
		cfg:     cfg,
		log:     log,
		console: console,
	}, nil
}

func (k *Kernel) Config() *config.Config {
	return k.cfg
}

func (k *Kernel) Logger() *logger.Logger {
	return k.log
}

func (k *Kernel) Command(cmds ...cli.Command) {
	for _, cmd := range cmds {
		if err := k.console.Register(cmd); err != nil {
			panic(fmt.Sprintf(
				"framework: register command %q: %v",
				cmd.Name(), err,
			))
		}
	}
}

func (k *Kernel) OnShutdown(fn func()) {
	k.cleanups = append(k.cleanups, fn)
}

func (k *Kernel) Run(ctx context.Context, args []string) error {
	defer k.runCleanups()

	return k.console.Run(ctx, os.Stdin, os.Stdout, args)
}

func (k *Kernel) RunWith(
	ctx context.Context,
	in io.Reader,
	out io.Writer,
	args []string,
) error {
	defer k.runCleanups()

	return k.console.Run(ctx, in, out, args)
}

func (k *Kernel) runCleanups() {
	for i := len(k.cleanups) - 1; i >= 0; i-- {
		k.cleanups[i]()
	}
}
