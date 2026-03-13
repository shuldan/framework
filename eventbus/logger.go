package eventbus

// Logger — интерфейс логгера для eventbus.
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

type noopLogger struct{}

func (noopLogger) Info(string, ...any)  {}
func (noopLogger) Error(string, ...any) {}
