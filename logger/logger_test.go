package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestNew_DefaultsToInfoJSON(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	log := NewWithWriter(&buf, Config{})
	log.Info("test message", "key", "val")
	output := buf.String()
	assertContains(t, output, `"level":"INFO"`)
	assertContains(t, output, `"msg":"test message"`)
	assertContains(t, output, `"key":"val"`)
}

func TestNew_TextFormat(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	log := NewWithWriter(&buf, Config{Format: "text"})
	log.Info("hello")
	output := buf.String()
	assertContains(t, output, "level=INFO")
	assertContains(t, output, "hello")
}

func TestLogger_LevelFiltering(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, cfgLevel, logMethod, message string
		visible                            bool
	}{
		{"debug at info", "info", "debug", "dbg", false},
		{"info at info", "info", "info", "inf", true},
		{"warn at info", "info", "warn", "wrn", true},
		{"error at info", "info", "error", "err", true},
		{"debug at debug", "debug", "debug", "dbg2", true},
		{"info at error", "error", "info", "inf2", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			log := NewWithWriter(&buf, Config{Level: tt.cfgLevel})
			callMethod(log, tt.logMethod, tt.message)
			output := buf.String()
			if tt.visible && !strings.Contains(output, tt.message) {
				t.Errorf("expected %q in output", tt.message)
			}
			if !tt.visible && strings.Contains(output, tt.message) {
				t.Errorf("did not expect %q in output", tt.message)
			}
		})
	}
}

func TestLogger_With_AddsContext(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	log := NewWithWriter(&buf, Config{})
	child := log.With("module", "auth")
	child.Info("access granted")
	output := buf.String()
	assertContains(t, output, `"module":"auth"`)
	assertContains(t, output, "access granted")
}

func TestLogger_ParseLevel_AllValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		level    string
		expected string
	}{
		{"debug", "DEBUG"}, {"info", "INFO"}, {"warn", "WARN"},
		{"warning", "WARN"}, {"error", "ERROR"}, {"unknown", "INFO"}, {"", "INFO"},
	}
	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			log := NewWithWriter(&buf, Config{Level: tt.level})
			log.Error("test")
			assertContains(t, buf.String(), `"level":"ERROR"`)
		})
	}
}

func TestNew_StdoutDefault(t *testing.T) {
	t.Parallel()
	log := New(Config{})
	if log == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestNew_StderrOutput(t *testing.T) {
	t.Parallel()
	log := New(Config{Output: "stderr"})
	if log == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestParseOutput_Stderr(t *testing.T) {
	t.Parallel()
	w := parseOutput("stderr")
	if w == nil {
		t.Fatal("expected non-nil writer for stderr")
	}
}

func TestParseOutput_Stdout(t *testing.T) {
	t.Parallel()
	w := parseOutput("stdout")
	if w == nil {
		t.Fatal("expected non-nil writer for stdout")
	}
}

func TestParseOutput_Unknown(t *testing.T) {
	t.Parallel()
	w := parseOutput("something")
	if w == nil {
		t.Fatal("expected non-nil writer for unknown output")
	}
}

func callMethod(log *Logger, method, msg string) {
	switch method {
	case "debug":
		log.Debug(msg)
	case "info":
		log.Info(msg)
	case "warn":
		log.Warn(msg)
	case "error":
		log.Error(msg)
	}
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected %q in output:\n%s", needle, haystack)
	}
}
