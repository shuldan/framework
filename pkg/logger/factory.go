package logger

import (
	"log/slog"
	"os"

	"github.com/shuldan/framework/pkg/contracts"
)

func NewLogger(opts ...Option) (contracts.Logger, error) {
	cfg := &config{
		level:     slog.LevelInfo,
		json:      false,
		addSource: false,
		writer:    os.Stdout,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.replaceAttr == nil {
		WithDefaultReplaceAttr()(cfg)
	}

	handlerOpts := &slog.HandlerOptions{
		Level:       cfg.level,
		AddSource:   cfg.addSource,
		ReplaceAttr: cfg.replaceAttr,
	}

	var handler slog.Handler
	if cfg.json {
		handler = slog.NewJSONHandler(cfg.writer, handlerOpts)
	} else {
		isColored := cfg.wantColor && isTerminal(cfg.writer)
		handler = newTextHandler(cfg.writer, isColored, handlerOpts.ReplaceAttr)
	}

	return &sLogger{Logger: slog.New(handler)}, nil
}

func NewModule(opts ...Option) contracts.AppModule {
	return &module{opts: opts}
}
