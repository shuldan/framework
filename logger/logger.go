package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

type Logger struct {
	slog *slog.Logger
}

type Config struct {
	Level  string // debug, info, warn, error
	Format string // json, text
	Output string // stdout, stderr
}

func New(cfg Config) *Logger {
	w := parseOutput(cfg.Output)

	return newLogger(w, cfg)
}

func NewWithWriter(w io.Writer, cfg Config) *Logger {
	return newLogger(w, cfg)
}

func (l *Logger) Debug(msg string, args ...any) {
	l.slog.Debug(msg, args...)
}

func (l *Logger) Info(msg string, args ...any) {
	l.slog.Info(msg, args...)
}

func (l *Logger) Warn(msg string, args ...any) {
	l.slog.Warn(msg, args...)
}

func (l *Logger) Error(msg string, args ...any) {
	l.slog.Error(msg, args...)
}

func (l *Logger) With(args ...any) *Logger {
	return &Logger{slog: l.slog.With(args...)}
}

func newLogger(w io.Writer, cfg Config) *Logger {
	level := parseLevel(cfg.Level)
	handler := newHandler(cfg.Format, w, level)

	return &Logger{slog: slog.New(handler)}
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func parseOutput(s string) io.Writer {
	if strings.ToLower(s) == "stderr" {
		return os.Stderr
	}

	return os.Stdout
}

func newHandler(
	format string, w io.Writer, level slog.Level,
) slog.Handler {
	opts := &slog.HandlerOptions{Level: level}

	if strings.ToLower(format) == "text" {
		return slog.NewTextHandler(w, opts)
	}

	return slog.NewJSONHandler(w, opts)
}
