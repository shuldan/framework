package eventbus

import "testing"

func TestWithSource(t *testing.T) {
	t.Parallel()
	cfg := &outboundConfig{}
	WithSource("order-service")(cfg)
	if cfg.source != "order-service" {
		t.Errorf("expected 'order-service', got %q", cfg.source)
	}
}

func TestWithSource_Empty(t *testing.T) {
	t.Parallel()
	cfg := &outboundConfig{}
	WithSource("")(cfg)
	if cfg.source != "" {
		t.Errorf("expected empty, got %q", cfg.source)
	}
}
