package logger

import (
	"log/slog"
	"testing"
)

func TestGetLevelName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		level    slog.Leveler
		expected string
	}{
		{levelTrace, "TRACE"},
		{levelCritical, "CRITICAL"},
		{slog.LevelDebug, "DEBUG"},
		{slog.LevelInfo, "INFO"},
		{slog.LevelWarn, "WARN"},
		{slog.LevelError, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			name := getLevelName(tt.level)
			if name != tt.expected {
				t.Errorf("getLevelName(%v) = %q, want %q", tt.level, name, tt.expected)
			}
		})
	}
}
