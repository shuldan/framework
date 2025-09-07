package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/shuldan/framework/pkg/contracts"
)

var oddArgsWarning sync.Once

type sLogger struct {
	*slog.Logger
}

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

	var handler slog.Handler
	if cfg.json {
		handlerOpts := &slog.HandlerOptions{
			Level:       cfg.level,
			AddSource:   cfg.addSource,
			ReplaceAttr: cfg.replaceAttr,
		}
		handler = slog.NewJSONHandler(cfg.writer, handlerOpts)
	} else {
		isColored := cfg.wantColor && isTerminal(cfg.writer)
		handler = newTextHandler(cfg.writer, isColored, cfg.replaceAttr, cfg.level)
	}

	return &sLogger{Logger: slog.New(handler)}, nil
}

func (l *sLogger) Trace(msg string, args ...any) {
	l.LogAttrs(context.Background(), levelTrace, msg, convertArgs(args)...)
}

func (l *sLogger) Debug(msg string, args ...any) {
	l.LogAttrs(context.Background(), slog.LevelDebug, msg, convertArgs(args)...)
}

func (l *sLogger) Info(msg string, args ...any) {
	l.LogAttrs(context.Background(), slog.LevelInfo, msg, convertArgs(args)...)
}

func (l *sLogger) Warn(msg string, args ...any) {
	l.LogAttrs(context.Background(), slog.LevelWarn, msg, convertArgs(args)...)
}

func (l *sLogger) Error(msg string, args ...any) {
	l.LogAttrs(context.Background(), slog.LevelError, msg, convertArgs(args)...)
}

func (l *sLogger) Critical(msg string, args ...any) {
	l.LogAttrs(context.Background(), levelCritical, msg, convertArgs(args)...)
}

func (l *sLogger) With(args ...any) contracts.Logger {
	return &sLogger{
		Logger: l.Logger.With(args...),
	}
}

func convertArgs(args []any) []slog.Attr {
	if len(args)%2 != 0 {
		oddArgsWarning.Do(func() {
			slog.Warn("logger called with odd number of args", slog.Any("args", args))
		})
	}

	var attrs []slog.Attr
	for i := 0; i < len(args); i += 2 {
		var key string
		if i+1 >= len(args) {
			keyVal := args[i]
			attrs = append(attrs, slog.Any("MISSING_KEY", keyVal))
			break
		}
		if k, ok := args[i].(string); ok {
			key = k
		} else {
			key = fmt.Sprintf("NON_STRING_KEY_%T", args[i])
		}
		attrs = append(attrs, slog.Any(key, args[i+1]))
	}
	return attrs
}
