package framework

import (
	"github.com/shuldan/cli"
	"github.com/shuldan/config"

	"github.com/shuldan/framework/logger"
)

func buildConfig(o *kernelOptions) (*config.Config, error) {
	if o.config != nil {
		return o.config, nil
	}

	opts := buildConfigOpts(o)
	if len(opts) == 0 {
		return config.FromMap(make(map[string]any)), nil
	}

	return config.New(opts...)
}

func buildConfigOpts(o *kernelOptions) []config.Option {
	capacity := len(o.configFiles) + 1
	opts := make([]config.Option, 0, capacity)

	opts = append(opts, buildFileOpts(o)...)

	if o.envPrefix != "" {
		loader := config.FromEnv(o.envPrefix).WithAutoTypeParse()
		opts = append(opts, loader)
	}

	return opts
}

func buildFileOpts(o *kernelOptions) []config.Option {
	if len(o.configFiles) == 0 {
		return nil
	}

	if o.profileEnvVar != "" {
		return buildProfileOpts(o)
	}

	opts := make([]config.Option, 0, len(o.configFiles))
	for _, file := range o.configFiles {
		opts = append(opts, config.FromYAML(file))
	}

	return opts
}

func buildProfileOpts(o *kernelOptions) []config.Option {
	opts := make([]config.Option, 0, len(o.configFiles))
	for _, file := range o.configFiles {
		opt := config.WithProfileFromEnv(file, o.profileEnvVar)
		opts = append(opts, opt)
	}

	return opts
}

func buildLogger(
	cfg *config.Config, o *kernelOptions,
) *logger.Logger {
	if o.logger != nil {
		return o.logger
	}

	return logger.New(logger.Config{
		Level:  cfg.GetString("log.level", "info"),
		Format: cfg.GetString("log.format", "json"),
		Output: cfg.GetString("log.output", "stdout"),
	})
}

func buildConsole(
	cfg *config.Config,
) *cli.Console {
	consoleOpts := []cli.ConsoleOption{
		cli.WithName(cfg.GetString("app.name", "app")),
	}

	version := cfg.GetString("app.version")
	if version != "" {
		consoleOpts = append(
			consoleOpts,
			cli.WithVersion(version),
		)
	}

	return cli.New(consoleOpts...)
}
