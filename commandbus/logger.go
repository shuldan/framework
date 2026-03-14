package commandbus

// Logger — интерфейс логгера для commandbus.
type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
}

type noopLogger struct{}

func (noopLogger) Info(string, ...any)  {}
func (noopLogger) Warn(string, ...any)  {}
func (noopLogger) Error(string, ...any) {}
func (noopLogger) Debug(string, ...any) {}

func ensureLogger(log Logger) Logger {
	if log == nil {
		return noopLogger{}
	}

	return log
}
