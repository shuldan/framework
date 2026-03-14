package commandbus

import (
	"testing"
)

func TestBuildOpts_AsyncMode(t *testing.T) {
	t.Parallel()
	cfg := Config{Async: true}
	opts := buildOpts(cfg)
	if len(opts) == 0 {
		t.Fatal("expected at least one option")
	}
}

func TestBuildOpts_SyncMode(t *testing.T) {
	t.Parallel()
	cfg := Config{Async: false}
	opts := buildOpts(cfg)
	if len(opts) == 0 {
		t.Fatal("expected at least one option")
	}
}

func TestBuildOpts_Workers(t *testing.T) {
	t.Parallel()
	cfg := Config{Async: true, Workers: 4}
	opts := buildOpts(cfg)
	if len(opts) < 2 {
		t.Fatalf("expected at least 2 options, got %d", len(opts))
	}
}

func TestBuildOpts_ZeroWorkers(t *testing.T) {
	t.Parallel()
	cfg := Config{Async: true, Workers: 0}
	opts := buildOpts(cfg)
	found := false
	for range opts {
		found = true
	}
	if !found {
		t.Fatal("expected options")
	}
}

func TestBuildOpts_BufferSize(t *testing.T) {
	t.Parallel()
	cfg := Config{Async: true, BufferSize: 100}
	opts := buildOpts(cfg)
	if len(opts) < 2 {
		t.Fatalf("expected at least 2 options, got %d", len(opts))
	}
}

func TestBuildOpts_ZeroBufferSize(t *testing.T) {
	t.Parallel()
	cfg := Config{Async: true, BufferSize: 0}
	opts := buildOpts(cfg)
	if len(opts) == 0 {
		t.Fatal("expected options")
	}
}

func TestBuildOpts_Ordered(t *testing.T) {
	t.Parallel()
	cfg := Config{Async: true, Ordered: true, Workers: 2}
	opts := buildOpts(cfg)
	if len(opts) < 3 {
		t.Fatalf("expected at least 3 options, got %d", len(opts))
	}
}

func TestBuildOpts_AllOptions(t *testing.T) {
	t.Parallel()
	cfg := Config{Async: true, Workers: 8, BufferSize: 256, Ordered: true}
	opts := buildOpts(cfg)
	if len(opts) != 4 {
		t.Fatalf("expected 4 options, got %d", len(opts))
	}
}

func TestBuildOpts_SyncWithNoExtras(t *testing.T) {
	t.Parallel()
	cfg := Config{Async: false, Workers: 0, BufferSize: 0, Ordered: false}
	opts := buildOpts(cfg)
	if len(opts) != 1 {
		t.Fatalf("expected 1 option (sync mode), got %d", len(opts))
	}
}
