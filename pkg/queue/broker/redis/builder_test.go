package redis

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()
	config := defaultConfig()

	if config.streamKeyFormat != "stream:%s" {
		t.Errorf("expected 'stream:%%s', got %q", config.streamKeyFormat)
	}
	if config.consumerGroup != "consumers" {
		t.Errorf("expected 'consumers', got %q", config.consumerGroup)
	}
	if config.processingTimeout != 30*time.Second {
		t.Errorf("expected 30s, got %v", config.processingTimeout)
	}
	if config.claimInterval != 1*time.Second {
		t.Errorf("expected 1s, got %v", config.claimInterval)
	}
	if config.maxClaimBatch != 10 {
		t.Errorf("expected 10, got %d", config.maxClaimBatch)
	}
	if config.blockTimeout != 500*time.Millisecond {
		t.Errorf("expected 500ms, got %v", config.blockTimeout)
	}
	if config.maxStreamLength != 0 {
		t.Errorf("expected 0, got %d", config.maxStreamLength)
	}
	if !config.approximateTrim {
		t.Error("expected approximateTrim to be true")
	}
	if !config.enableClaim {
		t.Error("expected enableClaim to be true")
	}
	if config.consumerPrefix != "" {
		t.Errorf("expected empty string, got %q", config.consumerPrefix)
	}
}

func TestWithStreamKeyFormat(t *testing.T) {
	t.Parallel()
	config := &config{}
	WithStreamKeyFormat("custom:%s")(config)
	if config.streamKeyFormat != "custom:%s" {
		t.Errorf("expected 'custom:%%s', got %q", config.streamKeyFormat)
	}
}

func TestWithConsumerGroup(t *testing.T) {
	t.Parallel()
	config := &config{}
	WithConsumerGroup("test-group")(config)
	if config.consumerGroup != "test-group" {
		t.Errorf("expected 'test-group', got %q", config.consumerGroup)
	}
}

func TestWithProcessingTimeout(t *testing.T) {
	t.Parallel()
	config := &config{}
	timeout := 45 * time.Second
	WithProcessingTimeout(timeout)(config)
	if config.processingTimeout != timeout {
		t.Errorf("expected %v, got %v", timeout, config.processingTimeout)
	}
}

func TestWithClaimInterval(t *testing.T) {
	t.Parallel()
	config := &config{}
	interval := 2 * time.Second
	WithClaimInterval(interval)(config)
	if config.claimInterval != interval {
		t.Errorf("expected %v, got %v", interval, config.claimInterval)
	}
}

func TestWithMaxClaimBatch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    int
		expected int
	}{
		{5, 5},
		{0, 0},
		{-1, 0},
		{100, 100},
	}

	for _, tt := range tests {
		config := &config{}
		WithMaxClaimBatch(tt.input)(config)
		if config.maxClaimBatch != tt.expected {
			t.Errorf("input %d: expected %d, got %d", tt.input, tt.expected, config.maxClaimBatch)
		}
	}
}

func TestWithBlockTimeout(t *testing.T) {
	t.Parallel()
	config := &config{}
	timeout := 1 * time.Second
	WithBlockTimeout(timeout)(config)
	if config.blockTimeout != timeout {
		t.Errorf("expected %v, got %v", timeout, config.blockTimeout)
	}
}

func TestWithMaxStreamLength(t *testing.T) {
	t.Parallel()
	config := &config{}
	WithMaxStreamLength(1000)(config)
	if config.maxStreamLength != 1000 {
		t.Errorf("expected 1000, got %d", config.maxStreamLength)
	}
}

func TestWithApproximateTrimming(t *testing.T) {
	t.Parallel()
	config := &config{}
	WithApproximateTrimming(false)(config)
	if config.approximateTrim {
		t.Error("expected approximateTrim to be false")
	}
	WithApproximateTrimming(true)(config)
	if !config.approximateTrim {
		t.Error("expected approximateTrim to be true")
	}
}

func TestWithClaim(t *testing.T) {
	t.Parallel()
	config := &config{}
	WithClaim(false)(config)
	if config.enableClaim {
		t.Error("expected enableClaim to be false")
	}
	WithClaim(true)(config)
	if !config.enableClaim {
		t.Error("expected enableClaim to be true")
	}
}

func TestWithConsumerPrefix(t *testing.T) {
	t.Parallel()
	config := &config{}
	WithConsumerPrefix("test-prefix")(config)
	if config.consumerPrefix != "test-prefix" {
		t.Errorf("expected 'test-prefix', got %q", config.consumerPrefix)
	}
}
