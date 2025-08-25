package queue

import (
	"errors"
	"testing"
)

type TestJob struct {
	Data string
}

func TestNew_ValidType(t *testing.T) {
	broker := &mockBroker{}
	_, err := New[*TestJob](broker)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestNew_InvalidType_NonPtr(t *testing.T) {
	broker := &mockBroker{}
	_, err := New[int](broker)
	if err == nil {
		t.Fatal("expected error for non-ptr type")
	}
	if !errors.Is(err, ErrInvalidJobType) {
		t.Errorf("expected ErrInvalidJobType, got %v", err)
	}
}

func TestNew_InvalidType_NilPtr(t *testing.T) {
	_, err := New[*int](nil)
	if err == nil {
		t.Fatal("expected error for nil ptr type")
	}
	if !errors.Is(err, ErrInvalidJobType) {
		t.Errorf("expected ErrInvalidJobType, got %v", err)
	}
}
