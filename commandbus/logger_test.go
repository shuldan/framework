package commandbus

import (
	"testing"
)

func TestEnsureLogger_Nil(t *testing.T) {
	t.Parallel()
	log := ensureLogger(nil)
	if log == nil {
		t.Fatal("expected non-nil logger")
	}
	if _, ok := log.(noopLogger); !ok {
		t.Errorf("expected noopLogger, got %T", log)
	}
}

func TestEnsureLogger_NonNil(t *testing.T) {
	t.Parallel()
	custom := newRecordingLogger()
	log := ensureLogger(custom)
	if log != custom {
		t.Errorf("expected custom logger to be returned")
	}
}

func TestNoopLogger_Methods(t *testing.T) {
	t.Parallel()
	var nl noopLogger
	nl.Info("test")
	nl.Warn("test")
	nl.Error("test")
	nl.Debug("test")
}
