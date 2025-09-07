package events

import "testing"

func TestWithPanicHandler(t *testing.T) {
	t.Parallel()

	handler := &mockPanicHandler{}
	cfg := &eventBusConfig{}

	opt := WithPanicHandler(handler)
	opt(cfg)

	if cfg.panicHandler != handler {
		t.Error("panic handler not set correctly")
	}
}

func TestWithErrorHandler(t *testing.T) {
	t.Parallel()

	handler := &mockErrorHandler{}
	cfg := &eventBusConfig{}

	opt := WithErrorHandler(handler)
	opt(cfg)

	if cfg.errorHandler != handler {
		t.Error("error handler not set correctly")
	}
}

func TestWithAsyncMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value bool
	}{
		{"async true", true},
		{"async false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &eventBusConfig{}
			opt := WithAsyncMode(tt.value)
			opt(cfg)

			if cfg.asyncMode != tt.value {
				t.Errorf("expected asyncMode %v, got %v", tt.value, cfg.asyncMode)
			}
		})
	}
}

func TestWithWorkerCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{"positive count", 5, 5},
		{"zero count", 0, 1},
		{"negative count", -5, 1},
		{"one", 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &eventBusConfig{}
			opt := WithWorkerCount(tt.input)
			opt(cfg)

			if cfg.workerCount != tt.expected {
				t.Errorf("expected workerCount %d, got %d", tt.expected, cfg.workerCount)
			}
		})
	}
}
