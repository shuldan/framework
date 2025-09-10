package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"golang.org/x/term"
)

type textHandler struct {
	writer      io.Writer
	attrs       []slog.Attr
	groups      []string
	isColored   bool
	replaceAttr func(groups []string, a slog.Attr) slog.Attr
	level       slog.Level
}

func newTextHandler(
	writer io.Writer,
	isColored bool,
	replaceAttr func(groups []string, a slog.Attr) slog.Attr,
	level slog.Level,
) slog.Handler {
	return &textHandler{
		writer:      writer,
		isColored:   isColored,
		replaceAttr: replaceAttr,
		level:       level,
	}
}

func (h *textHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *textHandler) Handle(_ context.Context, r slog.Record) error {
	levelStr := getLevelName(r.Level)

	if h.replaceAttr != nil {
		levelAttr := slog.String(slog.LevelKey, levelStr)
		levelAttr = h.replaceAttr(h.groups, levelAttr)
		levelStr = levelAttr.Value.String()
	}

	var l string
	if h.isColored && isTerminal(h.writer) {
		l = colorize(levelStr, r.Level)
	} else {
		l = levelStr
	}

	_, _ = fmt.Fprintf(h.writer, "%s %s", l, r.Message)

	r.Attrs(func(a slog.Attr) bool {
		if h.replaceAttr != nil {
			a = h.replaceAttr(h.groups, a)
		}
		if a.Key == "" || a.Equal(slog.Attr{}) {
			return true
		}
		_, _ = fmt.Fprintf(h.writer, " %s=%q", a.Key, a.Value)
		return true
	})

	for _, a := range h.attrs {
		if h.replaceAttr != nil {
			a = h.replaceAttr(h.groups, a)
		}
		if a.Key == "" || a.Equal(slog.Attr{}) {
			continue
		}
		_, _ = fmt.Fprintf(h.writer, " %s=%q", a.Key, a.Value)
	}

	_, _ = fmt.Fprintln(h.writer)
	return nil
}

func (h *textHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)
	return &textHandler{
		writer: h.writer,
		attrs:  newAttrs,
		groups: h.groups,
	}
}

func (h *textHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	newGroups := make([]string, len(h.groups), len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups = append(newGroups, name)
	return &textHandler{
		writer: h.writer,
		attrs:  h.attrs,
		groups: newGroups,
	}
}

func colorize(levelStr string, level slog.Level) string {
	const (
		reset  = "\033[0m"
		blue   = "\033[34m"
		cyan   = "\033[36m"
		green  = "\033[32m"
		yellow = "\033[33m"
		red    = "\033[31m"
		white  = "\033[37m"
		redBg  = "\033[41m"
	)

	switch level {
	case levelTrace:
		return cyan + levelStr + reset
	case slog.LevelDebug:
		return blue + levelStr + reset
	case slog.LevelInfo:
		return green + levelStr + reset
	case slog.LevelWarn:
		return yellow + levelStr + reset
	case slog.LevelError:
		return red + levelStr + reset
	case levelCritical:
		return redBg + white + levelStr + reset
	default:
		switch {
		case level < slog.LevelInfo:
			return cyan + levelStr + reset
		case level < slog.LevelWarn:
			return green + levelStr + reset
		case level < slog.LevelError:
			return yellow + levelStr + reset
		default:
			return red + levelStr + reset
		}
	}
}

func isTerminal(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}
