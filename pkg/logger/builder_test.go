package logger

import (
	"bytes"
	"io"
	"log/slog"
	"testing"
)

func TestWithLevel(t *testing.T) {
	cfg := &config{}
	WithLevel(slog.LevelDebug)(cfg)
	if cfg.level != slog.LevelDebug {
		t.Errorf("WithLevel: got %v, want %v", cfg.level, slog.LevelDebug)
	}
}

func TestWithJSON(t *testing.T) {
	cfg := &config{}
	WithJSON()(cfg)
	if !cfg.json {
		t.Error("WithJSON: expected json=true")
	}
}

func TestWithText(t *testing.T) {
	cfg := &config{json: true}
	WithText()(cfg)
	if cfg.json {
		t.Error("WithText: expected json=false")
	}
}

func TestWithSource(t *testing.T) {
	cfg := &config{}
	WithSource()(cfg)
	if !cfg.addSource {
		t.Error("WithSource: expected addSource=true")
	}
}

func TestWithWriter(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &config{}
	WithWriter(buf)(cfg)
	if cfg.writer != buf {
		t.Error("WithWriter: writer not set correctly")
	}

	WithWriter(nil)(cfg)
	if cfg.writer != io.Discard {
		t.Error("WithWriter(nil) should use io.Discard")
	}
}

func TestWithReplaceAttr(t *testing.T) {
	cfg := &config{}
	WithReplaceAttr(func(groups []string, a slog.Attr) slog.Attr {
		return slog.String("test", "replaced")
	})(cfg)
	if cfg.replaceAttr == nil {
		t.Error("WithReplaceAttr: replaceAttr not set")
	}
}

func TestWithLevelNames(t *testing.T) {
	names := map[slog.Leveler]string{slog.LevelDebug: "DBG"}
	cfg := &config{}
	WithLevelNames(names)(cfg)

	if cfg.replaceAttr == nil {
		t.Fatal("WithLevelNames: replaceAttr should be set")
	}

	attr := cfg.replaceAttr(nil, slog.Any(slog.LevelKey, slog.LevelDebug))

	if attr.Key != slog.LevelKey {
		t.Errorf("Expected key %q, got %q", slog.LevelKey, attr.Key)
	}
	if attr.Value.String() != "DBG" {
		t.Errorf("WithLevelNames: got %q, want %q", attr.Value.String(), "DBG")
	}
}

func TestWithColor(t *testing.T) {
	cfg := &config{}
	WithColor()(cfg)
	if !cfg.wantColor {
		t.Error("WithColor: wantColor not set")
	}
}
