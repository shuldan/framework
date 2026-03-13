package eventbus

import "testing"

func TestWithServiceName(t *testing.T) {
	t.Parallel()
	cfg := &inboundConfig{}
	WithServiceName("order-svc")(cfg)
	if cfg.service != "order-svc" {
		t.Errorf("expected 'order-svc', got %q", cfg.service)
	}
}

func TestWithServiceName_Empty(t *testing.T) {
	t.Parallel()
	cfg := &inboundConfig{}
	WithServiceName("")(cfg)
	if cfg.service != "" {
		t.Errorf("expected empty, got %q", cfg.service)
	}
}
