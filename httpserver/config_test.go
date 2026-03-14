package httpserver

import (
	"testing"
	"time"
)

func TestConfig_WithDefaults_AllZero(t *testing.T) {
	t.Parallel()
	cfg := Config{}.withDefaults()
	if cfg.ReadTimeout != 15*time.Second {
		t.Errorf("expected 15s read timeout, got %v", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 15*time.Second {
		t.Errorf("expected 15s write timeout, got %v", cfg.WriteTimeout)
	}
	if cfg.IdleTimeout != 60*time.Second {
		t.Errorf("expected 60s idle timeout, got %v", cfg.IdleTimeout)
	}
}

func TestConfig_WithDefaults_CustomValues(t *testing.T) {
	t.Parallel()
	cfg := Config{
		Port:         9090,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}.withDefaults()
	if cfg.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Port)
	}
	if cfg.ReadTimeout != 5*time.Second {
		t.Errorf("expected 5s, got %v", cfg.ReadTimeout)
	}
}
