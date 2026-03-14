package commandbus

import (
	"testing"
)

func TestWithListenerServiceName(t *testing.T) {
	t.Parallel()
	cfg := &replyListenerConfig{}
	WithListenerServiceName("my-service")(cfg)
	if cfg.serviceName != "my-service" {
		t.Errorf("expected %q, got %q", "my-service", cfg.serviceName)
	}
}

func TestWithListenerServiceName_Empty(t *testing.T) {
	t.Parallel()
	cfg := &replyListenerConfig{}
	WithListenerServiceName("")(cfg)
	if cfg.serviceName != "" {
		t.Errorf("expected empty, got %q", cfg.serviceName)
	}
}
