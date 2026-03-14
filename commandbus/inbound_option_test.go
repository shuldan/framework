package commandbus

import (
	"testing"
	"time"
)

func TestWithIdempotencyStore_Option(t *testing.T) {
	t.Parallel()
	store := newStubIdempotencyStore()
	cfg := &receiverConfig{}
	WithIdempotencyStore(store)(cfg)
	if cfg.idemStore != store {
		t.Error("expected store to be set")
	}
}

func TestWithIdempotencyTTL_Option(t *testing.T) {
	t.Parallel()
	cfg := &receiverConfig{}
	WithIdempotencyTTL(2 * time.Hour)(cfg)
	if cfg.idemTTL != 2*time.Hour {
		t.Errorf("expected 2h, got %v", cfg.idemTTL)
	}
}

func TestWithCommandIdempotencyTTL_Option(t *testing.T) {
	t.Parallel()
	entry := &inboundEntry{idemTTL: time.Hour}
	WithCommandIdempotencyTTL(30 * time.Minute)(entry)
	if entry.idemTTL != 30*time.Minute {
		t.Errorf("expected 30m, got %v", entry.idemTTL)
	}
}
