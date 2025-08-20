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
	handler := newTextHandler(buf, false, nil)

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
	handler := newTextHandler(buf, false, nil).WithAttrs([]slog.Attr{slog.String("service", "auth")})

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "login", 0)
	_ = handler.Handle(context.Background(), r)

	output := buf.String()
	if !strings.Contains(output, "service=\"auth\"") {
		t.Errorf("Expected service attr, got: %q", output)
	}
}

func TestTextHandler_WithGroup(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := newTextHandler(buf, false, nil).WithGroup("http")

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
	defer os.Remove(f.Name())
	defer f.Close()

	if isTerminal(f) {
		t.Error("isTerminal: temp file should not be terminal")
	}

	_ = isTerminal(os.Stdout)
}
