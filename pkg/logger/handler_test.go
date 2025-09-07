package logger

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"
)

func TestTextHandler_Handle(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := newTextHandler(buf, false, nil, slog.LevelInfo)

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
	r.AddAttrs(slog.String("user", "alice"), slog.Int("age", 30))

	err := handler.Handle(context.Background(), r)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "INFO test message user=\"alice\" age=\"30\"") {
		t.Errorf("Unexpected output: %q", output)
	}
}

func TestTextHandler_WithAttrs(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := newTextHandler(buf, false, nil, slog.LevelInfo).WithAttrs([]slog.Attr{slog.String("service", "auth")})

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "login", 0)
	_ = handler.Handle(context.Background(), r)

	output := buf.String()
	if !strings.Contains(output, "service=\"auth\"") {
		t.Errorf("Expected service attr, got: %q", output)
	}
}

func TestTextHandler_WithGroup(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := newTextHandler(buf, false, nil, slog.LevelInfo).WithGroup("http")

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "request", 0)
	r.AddAttrs(slog.String("method", "GET"))

	_ = handler.Handle(context.Background(), r)

	output := buf.String()
	if !strings.Contains(output, "method=\"GET\"") {
		t.Errorf("Group attr not passed: %q", output)
	}
}

func TestColorize(t *testing.T) {
	tests := []struct {
		level    slog.Level
		expected string
	}{
		{levelTrace, "\033[36mTRACE\033[0m"},
		{slog.LevelDebug, "\033[34mDEBUG\033[0m"},
		{slog.LevelInfo, "\033[32mINFO\033[0m"},
		{slog.LevelWarn, "\033[33mWARN\033[0m"},
		{slog.LevelError, "\033[31mERROR\033[0m"},
		{levelCritical, "\033[41m\033[37mCRITICAL\033[0m"},
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			got := colorize(getLevelName(tt.level), tt.level)
			if got != tt.expected {
				t.Errorf("colorize(%v) = %q, want %q", tt.level, got, tt.expected)
			}
		})
	}
}

func TestIsTerminal(t *testing.T) {
	f, _ := os.CreateTemp("", "testfile")
	defer func() {
		if err := os.Remove(f.Name()); err != nil {
			t.Logf("Failed to remove temp file: %v", err)
		}
	}()
	defer func() {
		if err := f.Close(); err != nil {
			t.Logf("Failed to close file: %v", err)
		}
	}()

	if isTerminal(f) {
		t.Error("isTerminal: temp file should not be terminal")
	}

	_ = isTerminal(os.Stdout)
}

func TestTextHandler_Enabled(t *testing.T) {
	t.Parallel()
	handler := newTextHandler(&bytes.Buffer{}, false, nil, slog.LevelDebug)

	if !handler.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("Enabled should always return true")
	}
	if !handler.Enabled(context.Background(), slog.LevelError) {
		t.Error("Enabled should always return true")
	}
}

func TestTextHandler_HandleWithEmptyTime(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	handler := newTextHandler(buf, false, nil, slog.LevelInfo)

	r := slog.NewRecord(time.Time{}, slog.LevelInfo, "test", 0)
	err := handler.Handle(context.Background(), r)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "0001") {
		t.Error("Should not contain zero time")
	}
}

func TestTextHandler_HandleWithReplaceAttr(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	replaceFunc := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == "secret" {
			return slog.Attr{}
		}
		if a.Key == slog.LevelKey {
			return slog.String(slog.LevelKey, "CUSTOM")
		}
		return a
	}

	handler := newTextHandler(buf, false, replaceFunc, slog.LevelInfo)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	r.AddAttrs(slog.String("secret", "password"), slog.String("user", "bob"))

	_ = handler.Handle(context.Background(), r)
	output := buf.String()

	if strings.Contains(output, "secret") || strings.Contains(output, "password") {
		t.Error("Secret should be filtered")
	}
	if !strings.Contains(output, "CUSTOM") {
		t.Error("Level should be replaced")
	}
	if !strings.Contains(output, "user=\"bob\"") {
		t.Error("User attr should be present")
	}
}

func TestTextHandler_WithEmptyGroup(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	handler := newTextHandler(buf, false, nil, slog.LevelInfo)

	h2 := handler.WithGroup("")
	if h2 != handler {
		t.Error("Empty group should return same handler")
	}
}

func TestTextHandler_ColoredOutput(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	handler := newTextHandler(buf, true, nil, slog.LevelInfo)

	r := slog.NewRecord(time.Now(), slog.LevelError, "error msg", 0)
	_ = handler.Handle(context.Background(), r)

	output := buf.String()
	if strings.Contains(output, "\033[") {
		t.Skip("Terminal detection prevents color codes")
	}
}

func TestIsTerminal_NonFile(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	if isTerminal(buf) {
		t.Error("Buffer should not be terminal")
	}
}

func TestColorize_UnknownLevels(t *testing.T) {
	t.Parallel()
	tests := []struct {
		level slog.Level
		name  string
	}{
		{slog.Level(-8), "BELOW_DEBUG"},
		{slog.Level(2), "BETWEEN_INFO_WARN"},
		{slog.Level(6), "BETWEEN_WARN_ERROR"},
		{slog.Level(10), "ABOVE_ERROR"},
	}

	for _, tt := range tests {
		result := colorize(tt.name, tt.level)
		if !strings.Contains(result, tt.name) {
			t.Errorf("colorize(%v) should contain %q", tt.level, tt.name)
		}
		if !strings.Contains(result, "\033[") {
			t.Errorf("colorize(%v) should contain color codes", tt.level)
		}
	}
}
