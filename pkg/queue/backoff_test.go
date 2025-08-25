package queue

import (
	"testing"
	"time"
)

func TestFixedBackoff_Delay(t *testing.T) {
	b := FixedBackoff{Duration: 500 * time.Millisecond}
	if b.Delay(1) != 500*time.Millisecond {
		t.Errorf("expected 500ms, got %v", b.Delay(1))
	}
	if b.Delay(10) != 500*time.Millisecond {
		t.Errorf("expected 500ms, got %v", b.Delay(10))
	}
}

func TestExponentialBackoff_Delay(t *testing.T) {
	b := ExponentialBackoff{Base: 100 * time.Millisecond, MaxDelay: 1 * time.Second}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 400 * time.Millisecond},
		{3, 800 * time.Millisecond},
		{4, 1 * time.Second},
		{5, 1 * time.Second},
	}

	for _, tt := range tests {
		if got := b.Delay(tt.attempt); got != tt.expected {
			t.Errorf("attempt %d: expected %v, got %v", tt.attempt, tt.expected, got)
		}
	}
}

func TestNoBackoff_Delay(t *testing.T) {
	var b NoBackoff
	if b.Delay(1) != 0 {
		t.Errorf("expected 0, got %v", b.Delay(1))
	}
	if b.Delay(100) != 0 {
		t.Errorf("expected 0, got %v", b.Delay(100))
	}
}
