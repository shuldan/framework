package logger

import (
	"log/slog"
	"strings"
	"time"

	"github.com/shuldan/framework/pkg/contracts"
)

type module struct{}

func NewModule() contracts.AppModule {
	return &module{}
}

func (m *module) Name() string {
	return contracts.LoggerModuleName
}

func (m *module) Register(container contracts.DIContainer) error {
	return container.Factory(
		contracts.LoggerModuleName,
		func(c contracts.DIContainer) (interface{}, error) {
			options := m.getLoggerOptions(c)
			return NewLogger(options...)
		},
	)
}

func (m *module) Start(ctx contracts.AppContext) error {
	if log, err := ctx.Container().Resolve(contracts.LoggerModuleName); err == nil {
		if logger, ok := log.(contracts.Logger); ok {
			logger.Info("Logging started",
				"app", ctx.AppName(),
				"version", ctx.Version(),
				"environment", ctx.Environment(),
			)
		}
	}
	return nil
}

func (m *module) Stop(ctx contracts.AppContext) error {
	if log, err := ctx.Container().Resolve(contracts.LoggerModuleName); err == nil {
		if logger, ok := log.(contracts.Logger); ok {
			logger.Info("Logging stopped",
				"app", ctx.AppName(),
				"uptime", time.Since(ctx.StartTime()),
			)
		}
	}
	return nil
}

func (m *module) getLoggerOptions(c contracts.DIContainer) []Option {
	var options []Option

	if configInst, err := c.Resolve(contracts.ConfigModuleName); err == nil {
		if cfg, ok := configInst.(contracts.Config); ok {
			if loggerCfg, ok := cfg.GetSub("logger"); ok {
				options = append(options, m.optionsFromFileConfig(loggerCfg)...)
			}
		}
	}

	if len(options) == 0 {
		options = append(options,
			WithLevel(slog.LevelInfo),
			WithText(),
			WithColor(),
			WithDefaultReplaceAttr(),
		)
	}

	return options
}

func (m *module) optionsFromFileConfig(cfg contracts.Config) []Option {
	var options []Option

	levelStr := cfg.GetString("level", "info")
	level := m.parseLevel(levelStr)
	options = append(options, WithLevel(level))

	format := cfg.GetString("format", "text")
	if strings.ToLower(format) == "json" {
		options = append(options, WithJSON())
	} else {
		options = append(options, WithText())
	}

	if cfg.GetBool("add_source", false) {
		options = append(options, WithSource())
	}

	if cfg.GetBool("color", true) {
		options = append(options, WithColor())
	}

	options = append(options, WithDefaultReplaceAttr())

	return options
}

func (m *module) parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "trace":
		return levelTrace
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "critical", "fatal":
		return levelCritical
	default:
		return slog.LevelInfo
	}
}
