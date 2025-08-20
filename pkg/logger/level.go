package logger

import "log/slog"

const (
	levelTrace    = slog.LevelDebug - 4
	levelCritical = slog.LevelError + 4
)

func getLevelName(level slog.Leveler) string {
	var levelNames = map[slog.Leveler]string{
		levelTrace:    "TRACE",
		levelCritical: "CRITICAL",
	}

	if name, ok := levelNames[level]; ok {
		return name
	}
	return level.Level().String()
}
