package queue

import (
	"errors"
	"testing"
)

func TestDefaultErrorHandler(t *testing.T) {
	h := &defaultErrorHandler{logger: nil}
	defer func() {
		if r := recover(); r != nil {
			t.Error("expected no panic, but got", r)
		}
	}()
	h.Handle("job", "consumer", errors.New("test"))
}

func TestDefaultPanicHandler(t *testing.T) {
	h := &defaultPanicHandler{logger: nil}
	defer func() {
		if r := recover(); r != nil {
			t.Error("expected no panic, but got", r)
		}
	}()
	h.Handle("job", "consumer", "panic", []byte("stack"))
}

func TestDefaultErrorHandler_WithNilJob(t *testing.T) {
	t.Parallel()
	h := &defaultErrorHandler{logger: nil}
	h.Handle(nil, "consumer", errors.New("test"))
}

func TestDefaultPanicHandler_WithNilJob(t *testing.T) {
	t.Parallel()
	h := &defaultPanicHandler{logger: nil}
	h.Handle(nil, "consumer", "panic", []byte("stack"))
}

func TestDefaultErrorHandler_WithLogger(t *testing.T) {
	t.Parallel()
	logger := &noOpLogger{}
	h := &defaultErrorHandler{logger: logger}
	h.Handle("job", "consumer", errors.New("test"))
}

func TestDefaultPanicHandler_WithLogger(t *testing.T) {
	t.Parallel()
	logger := &noOpLogger{}
	h := &defaultPanicHandler{logger: logger}
	h.Handle("job", "consumer", "panic", []byte("stack"))
}
