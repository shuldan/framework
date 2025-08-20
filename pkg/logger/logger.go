package logger

import (
	"fmt"
	"github.com/shuldan/framework/pkg/contracts"
	"log/slog"
	"sync"
)

var oddArgsWarning sync.Once

type sLogger struct {
	*slog.Logger
}

func (l *sLogger) Trace(msg string, args ...any) {
	l.Logger.LogAttrs(nil, levelTrace, msg, convertArgs(args)...)
}

func (l *sLogger) Debug(msg string, args ...any) {
	l.Logger.LogAttrs(nil, slog.LevelDebug, msg, convertArgs(args)...)
}

func (l *sLogger) Info(msg string, args ...any) {
	l.Logger.LogAttrs(nil, slog.LevelInfo, msg, convertArgs(args)...)
}

func (l *sLogger) Warn(msg string, args ...any) {
	l.Logger.LogAttrs(nil, slog.LevelWarn, msg, convertArgs(args)...)
}

func (l *sLogger) Error(msg string, args ...any) {
	l.Logger.LogAttrs(nil, slog.LevelError, msg, convertArgs(args)...)
}

func (l *sLogger) Critical(msg string, args ...any) {
	l.Logger.LogAttrs(nil, levelCritical, msg, convertArgs(args)...)
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
