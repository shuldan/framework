package events

import (
	"errors"
	"testing"

	"github.com/shuldan/framework/pkg/contracts"
)

const testListenerName = "test_listener"

type mockLogger struct {
	logs []logEntry
}

type logEntry struct {
	level  string
	msg    string
	fields []interface{}
}

func (m *mockLogger) Debug(msg string, fields ...interface{}) {
	m.logs = append(m.logs, logEntry{"debug", msg, fields})
}

func (m *mockLogger) Info(msg string, fields ...interface{}) {
	m.logs = append(m.logs, logEntry{"info", msg, fields})
}

func (m *mockLogger) Warn(msg string, fields ...interface{}) {
	m.logs = append(m.logs, logEntry{"warn", msg, fields})
}

func (m *mockLogger) Error(msg string, fields ...interface{}) {
	m.logs = append(m.logs, logEntry{"error", msg, fields})
}

func (m *mockLogger) Critical(msg string, fields ...interface{}) {
	m.logs = append(m.logs, logEntry{"critical", msg, fields})
}

func (m *mockLogger) Trace(msg string, args ...any) {
	m.logs = append(m.logs, logEntry{"trace", msg, args})
}

func (m *mockLogger) With(_ ...any) contracts.Logger {
	return m
}

func TestDefaultPanicHandler_WithLogger(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	handler := NewDefaultPanicHandler(logger)

	event := TestEvent{Message: "test"}
	listener := testListenerName
	panicValue := "test panic"
	stack := []byte("stack trace")

	handler.Handle(event, listener, panicValue, stack)

	if len(logger.logs) != 1 {
		t.Errorf("expected 1 log entry, got %d", len(logger.logs))
	}

	entry := logger.logs[0]
	if entry.level != "critical" {
		t.Errorf("expected critical level, got %s", entry.level)
	}

	if entry.msg != "event eventBus panic" {
		t.Errorf("expected 'event eventBus panic', got %s", entry.msg)
	}
}

func TestDefaultPanicHandler_WithoutLogger(t *testing.T) {
	t.Parallel()

	handler := NewDefaultPanicHandler(nil)

	event := TestEvent{Message: "test"}
	listener := testListenerName
	panicValue := "test panic"
	stack := []byte("stack trace")

	handler.Handle(event, listener, panicValue, stack)
}

func TestDefaultErrorHandler_WithLogger(t *testing.T) {
	t.Parallel()

	logger := &mockLogger{}
	handler := NewDefaultErrorHandler(logger)

	event := TestEvent{Message: "test"}
	listener := testListenerName
	err := errors.New("test error")

	handler.Handle(event, listener, err)

	if len(logger.logs) != 1 {
		t.Errorf("expected 1 log entry, got %d", len(logger.logs))
	}

	entry := logger.logs[0]
	if entry.level != "error" {
		t.Errorf("expected error level, got %s", entry.level)
	}

	if entry.msg != "event eventBus error" {
		t.Errorf("expected 'event eventBus error', got %s", entry.msg)
	}
}

func TestDefaultErrorHandler_WithoutLogger(t *testing.T) {
	t.Parallel()

	handler := NewDefaultErrorHandler(nil)

	event := TestEvent{Message: "test"}
	listener := testListenerName
	err := errors.New("test error")

	handler.Handle(event, listener, err)
}
