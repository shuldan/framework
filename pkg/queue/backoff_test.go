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

func TestFixedBackoff_ZeroDuration(t *testing.T) {
	t.Parallel()
	b := FixedBackoff{Duration: 0}
	if b.Delay(5) != 0 {
		t.Errorf("expected 0 for zero duration, got %v", b.Delay(5))
	}
}

func TestFixedBackoff_NegativeAttempt(t *testing.T) {
	t.Parallel()
	b := FixedBackoff{Duration: 100 * time.Millisecond}
	if b.Delay(-1) != 100*time.Millisecond {
		t.Errorf("expected 100ms for negative attempt, got %v", b.Delay(-1))
	}
}

func TestExponentialBackoff_NegativeAttempt(t *testing.T) {
	t.Parallel()
	b := ExponentialBackoff{Base: 100 * time.Millisecond, MaxDelay: 1 * time.Second}
	if b.Delay(-1) != 100*time.Millisecond {
		t.Errorf("expected base delay for negative attempt, got %v", b.Delay(-1))
	}
}

func TestExponentialBackoff_LargeAttempt(t *testing.T) {
	t.Parallel()
	b := ExponentialBackoff{Base: 100 * time.Millisecond, MaxDelay: 1 * time.Second}
	if b.Delay(63) != 1*time.Second {
		t.Errorf("expected max delay for attempt 63, got %v", b.Delay(63))
	}
	if b.Delay(100) != 1*time.Second {
		t.Errorf("expected max delay for attempt 100, got %v", b.Delay(100))
	}
}

func TestExponentialBackoff_Overflow(t *testing.T) {
	t.Parallel()
	b := ExponentialBackoff{Base: 1 * time.Hour, MaxDelay: 2 * time.Hour}
	if b.Delay(50) != 2*time.Hour {
		t.Errorf("expected max delay on overflow, got %v", b.Delay(50))
	}
}

func TestExponentialBackoff_ZeroBase(t *testing.T) {
	t.Parallel()
	b := ExponentialBackoff{Base: 0, MaxDelay: 1 * time.Second}
	if b.Delay(1) != 0 {
		t.Errorf("expected 0 for zero base, got %v", b.Delay(1))
	}
}

func TestNoBackoff_Various(t *testing.T) {
	t.Parallel()
	var b NoBackoff
	tests := []int{-10, 0, 1, 100, 1000}
	for _, attempt := range tests {
		if b.Delay(attempt) != 0 {
			t.Errorf("expected 0 for attempt %d, got %v", attempt, b.Delay(attempt))
		}
	}
}
