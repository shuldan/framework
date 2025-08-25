package queue

import (
	"errors"
	"testing"
)

func TestDefaultErrorHandler(t *testing.T) {
	h := &defaultErrorHandler{logger: nil}
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected log.Fatalln, but no panic")
		}
	}()
	h.Handle("job", "consumer", errors.New("test"))
}

func TestDefaultPanicHandler(t *testing.T) {
	h := &defaultPanicHandler{logger: nil}
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected log.Panicln, but no panic")
		}
	}()
	h.Handle("job", "consumer", "panic", []byte("stack"))
}
