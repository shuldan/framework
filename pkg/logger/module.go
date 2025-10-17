package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/shuldan/framework/pkg/contracts"
)

const ModuleName = "logger"

type module struct{}

func NewModule() contracts.AppModule {
	return &module{}
}

func (m *module) Name() string {
	return ModuleName
}

func (m *module) Register(container contracts.DIContainer) error {
	return container.Factory(
		reflect.TypeOf((*contracts.Logger)(nil)).Elem(),
		func(c contracts.DIContainer) (interface{}, error) {
			options, err := m.getLoggerOptions(c)
			if err != nil {
				return nil, fmt.Errorf("failed to get logger options: %w", err)
			}
			return NewLogger(options...)
		},
	)
}

func (m *module) Start(ctx contracts.AppContext) error {
	if log, err := ctx.Container().Resolve(reflect.TypeOf((*contracts.Logger)(nil)).Elem()); err == nil {
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
	if log, err := ctx.Container().Resolve(reflect.TypeOf((*contracts.Logger)(nil)).Elem()); err == nil {
		if logger, ok := log.(contracts.Logger); ok {
			logger.Info("Logging stopped",
				"app", ctx.AppName(),
				"uptime", time.Since(ctx.StartTime()),
			)
		}
	}
	return nil
}

func (m *module) getLoggerOptions(c contracts.DIContainer) ([]Option, error) {
	var options []Option

	if configInst, err := c.Resolve(reflect.TypeOf((*contracts.Config)(nil)).Elem()); err == nil {
		if cfg, ok := configInst.(contracts.Config); ok {
			if loggerCfg, ok := cfg.GetSub("logger"); ok {
				opts, err := m.optionsFromFileConfig(loggerCfg)
				if err != nil {
					return nil, err
				}
				options = append(options, opts...)
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

	return options, nil
}

func (m *module) optionsFromFileConfig(cfg contracts.Config) ([]Option, error) {
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

	output := cfg.GetString("output", "stdout")

	writer, err := m.getWriter(output, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create writer: %w", err)
	}
	options = append(options, WithWriter(writer))

	if cfg.GetBool("include_caller", false) {
		options = append(options, WithSource())
	}

	enableColors := cfg.GetBool("enable_colors", true)
	if enableColors && strings.ToLower(format) == "text" {
		options = append(options, WithColor())
	}

	options = append(options, WithDefaultReplaceAttr())

	return options, nil
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

func (m *module) getWriter(output string, cfg contracts.Config) (io.Writer, error) {
	switch strings.ToLower(output) {
	case "stdout":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	case "file":
		baseDir := cfg.GetString("base_dir", "./logs")
		filePath := cfg.GetString("file_path", "app.log")
		return m.createFileWriter(baseDir, filePath)
	default:
		return os.Stdout, nil
	}
}

func (m *module) createFileWriter(baseDir, filePath string) (io.Writer, error) {
	absBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve base directory: %w", err)
	}
	absPath := filepath.Join(absBaseDir, filePath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0750); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}
	file, err := os.OpenFile(filepath.Clean(absPath), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return file, nil
}
