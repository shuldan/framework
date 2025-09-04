package queue

import (
	"testing"
	"time"
)

func TestWithPrefix(t *testing.T) {
	t.Parallel()
	config := &queueConfig{}
	WithPrefix("test-prefix")(config)
	if config.prefix != "test-prefix" {
		t.Errorf("expected 'test-prefix', got %q", config.prefix)
	}
}

func TestWithDLQ(t *testing.T) {
	t.Parallel()
	config := &queueConfig{}
	WithDLQ(true)(config)
	if !config.dlqEnabled {
		t.Error("expected dlqEnabled to be true")
	}
	WithDLQ(false)(config)
	if config.dlqEnabled {
		t.Error("expected dlqEnabled to be false")
	}
}

func TestWithConcurrency(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    int
		expected int
	}{
		{5, 5},
		{0, 1},
		{-1, 1},
		{100, 100},
	}

	for _, tt := range tests {
		config := &queueConfig{}
		WithConcurrency(tt.input)(config)
		if config.concurrency != tt.expected {
			t.Errorf("input %d: expected %d, got %d", tt.input, tt.expected, config.concurrency)
		}
	}
}

func TestWithMaxRetries(t *testing.T) {
	t.Parallel()
	config := &queueConfig{}
	WithMaxRetries(3)(config)
	if config.maxRetries != 3 {
		t.Errorf("expected 3, got %d", config.maxRetries)
	}
}

func TestWithBackoff(t *testing.T) {
	t.Parallel()
	config := &queueConfig{}
	backoff := FixedBackoff{Duration: 100 * time.Millisecond}
	WithBackoff(backoff)(config)
	if config.backoff != backoff {
		t.Error("backoff was not set correctly")
	}
}

func TestWithErrorHandler(t *testing.T) {
	t.Parallel()
	config := &queueConfig{}
	handler := &defaultErrorHandler{}
	WithErrorHandler(handler)(config)
	if config.errorHandler != handler {
		t.Error("error handler was not set correctly")
	}
}

func TestWithPanicHandler(t *testing.T) {
	t.Parallel()
	config := &queueConfig{}
	handler := &defaultPanicHandler{}
	WithPanicHandler(handler)(config)
	if config.panicHandler != handler {
		t.Error("panic handler was not set correctly")
	}
}

func TestWithCounter(t *testing.T) {
	t.Parallel()
	config := &queueConfig{}
	counter := NoOpCounter{}
	WithCounter(counter)(config)
	if config.counter != counter {
		t.Error("counter was not set correctly")
	}
}
