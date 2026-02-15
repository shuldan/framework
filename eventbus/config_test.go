package eventbus

import (
	"testing"
)

func TestBuildOpts_SyncMode(t *testing.T) {
	t.Parallel()
	opts := buildOpts(Config{Async: false})
	if len(opts) < 1 {
		t.Fatal("expected at least 1 option")
	}
}

func TestBuildOpts_AsyncMode(t *testing.T) {
	t.Parallel()
	opts := buildOpts(Config{Async: true, Workers: 4, BufferSize: 100})
	if len(opts) < 3 {
		t.Fatalf("expected at least 3 options, got %d", len(opts))
	}
}

func TestBuildOpts_Ordered(t *testing.T) {
	t.Parallel()
	opts := buildOpts(Config{Ordered: true})
	if len(opts) < 2 {
		t.Fatalf("expected at least 2 options, got %d", len(opts))
	}
}

func TestBuildOpts_ZeroWorkersAndBuffer(t *testing.T) {
	t.Parallel()
	opts := buildOpts(Config{Async: false, Workers: 0, BufferSize: 0})
	if len(opts) < 1 {
		t.Fatal("expected at least 1 option")
	}
}

func TestBuildOpts_AllOptions(t *testing.T) {
	t.Parallel()
	opts := buildOpts(Config{
		Async: true, Workers: 2, BufferSize: 50, Ordered: true,
	})
	if len(opts) != 4 {
		t.Fatalf("expected 4 options, got %d", len(opts))
	}
}
