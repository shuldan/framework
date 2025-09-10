package logger

import (
	"io"
	"log/slog"
)

type Option func(*config)

type config struct {
	level       slog.Level
	json        bool
	addSource   bool
	writer      io.Writer
	replaceAttr func(groups []string, a slog.Attr) slog.Attr
	wantColor   bool
}

func WithReplaceAttr(f func(groups []string, a slog.Attr) slog.Attr) Option {
	return func(c *config) {
		c.replaceAttr = f
	}
}

func WithLevel(level slog.Level) Option {
	return func(c *config) {
		c.level = level
	}
}

func WithJSON() Option {
	return func(c *config) {
		c.json = true
	}
}

func WithText() Option {
	return func(c *config) {
		c.json = false
	}
}

func WithSource() Option {
	return func(c *config) {
		c.addSource = true
	}
}

func WithWriter(w io.Writer) Option {
	return func(c *config) {
		if w == nil {
			w = io.Discard
		}
		c.writer = w
	}
}

func WithLevelNames(names map[slog.Leveler]string) Option {
	return func(c *config) {
		prev := c.replaceAttr
		c.replaceAttr = func(groups []string, a slog.Attr) slog.Attr {
			if prev != nil {
				a = prev(groups, a)
			}
			if a.Key == slog.LevelKey {
				if level, ok := a.Value.Any().(slog.Level); ok {
					if label, exists := names[level]; exists {
						return slog.String(slog.LevelKey, label)
					}
					return slog.String(slog.LevelKey, getLevelName(level))
				}
			}
			return a
		}
	}
}

func WithColor() Option {
	return func(c *config) {
		c.wantColor = true
	}
}

func WithDefaultReplaceAttr() Option {
	return func(c *config) {
		prev := c.replaceAttr
		c.replaceAttr = func(groups []string, a slog.Attr) slog.Attr {
			if prev != nil {
				a = prev(groups, a)
				if a.Equal(slog.Attr{}) {
					return a
				}
			}
			if a.Key == slog.LevelKey {
				if level, ok := a.Value.Any().(slog.Level); ok {
					return slog.String(slog.LevelKey, getLevelName(level))
				}
			}
			return a
		}
	}
}
