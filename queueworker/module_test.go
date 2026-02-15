package queueworker

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestModule_Name(t *testing.T) {
	t.Parallel()
	m := NewModule(nil)
	if m.Name() != "queueworker" {
		t.Fatalf("expected 'queueworker', got %q", m.Name())
	}
}

func TestModule_Register(t *testing.T) {
	t.Parallel()
	m := NewModule(nil)
	m.Register(Registration{
		Name: "test",
		Run:  func(ctx context.Context) error { <-ctx.Done(); return nil },
	})
	if m.ConsumerCount() != 1 {
		t.Fatalf("expected 1 consumer, got %d", m.ConsumerCount())
	}
}

func TestModule_Lifecycle(t *testing.T) {
	m := NewModule(nil)
	var started atomic.Bool
	m.Register(Registration{
		Name: "worker-1",
		Run: func(ctx context.Context) error {
			started.Store(true)
			<-ctx.Done()
			return nil
		},
	})
	ctx := context.Background()
	if err := m.Init(ctx); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitFor(t, func() bool { return started.Load() })
	if err := m.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

func TestModule_NoRegistrations(t *testing.T) {
	t.Parallel()
	m := NewModule(nil)
	ctx := context.Background()
	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := m.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

func TestModule_ConsumerError_ReportedOnErrChannel(t *testing.T) {
	m := NewModule(nil)
	expected := errors.New("consumer crashed")
	m.Register(Registration{
		Name: "failing",
		Run:  func(_ context.Context) error { return expected },
	})
	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	select {
	case err := <-m.Err():
		if !errors.Is(err, expected) {
			t.Fatalf("expected %v, got %v", expected, err)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for error")
	}
	_ = m.Stop(context.Background())
}

func TestModule_ContextCanceled_NotReported(t *testing.T) {
	m := NewModule(nil)
	m.Register(Registration{
		Name: "normal",
		Run: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
	})
	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := m.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	select {
	case err := <-m.Err():
		t.Fatalf("unexpected error: %v", err)
	default:
	}
}

func TestModule_MultipleConsumers(t *testing.T) {
	m := NewModule(nil)
	var count atomic.Int32
	for i := range 3 {
		name := []string{"a", "b", "c"}[i]
		m.Register(Registration{
			Name: name,
			Run: func(ctx context.Context) error {
				count.Add(1)
				<-ctx.Done()
				return nil
			},
		})
	}
	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitFor(t, func() bool { return count.Load() == 3 })
	if err := m.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

func TestModule_EnsureLog_NonNil(t *testing.T) {
	t.Parallel()
	ml := &qwMockLogger{}
	l := ensureLog(ml)
	if l != ml {
		t.Fatal("expected same logger")
	}
}

func TestModule_NoopLogger(t *testing.T) {
	t.Parallel()
	l := noopLogger{}
	l.Info("test")
	l.Error("test")
}

func TestModule_WithLogger(t *testing.T) {
	ml := &qwMockLogger{}
	m := NewModule(ml)
	m.Register(Registration{
		Name: "logged",
		Run: func(ctx context.Context) error {
			<-ctx.Done()
			return nil
		},
	})
	_ = m.Start(context.Background())
	waitFor(t, func() bool { return ml.infoCount.Load() >= 3 })
	_ = m.Stop(context.Background())
	if ml.infoCount.Load() == 0 {
		t.Error("expected Info called")
	}
}

func TestModule_DeadlineExceeded_NotReported(t *testing.T) {
	m := NewModule(nil)
	m.Register(Registration{
		Name: "deadline",
		Run: func(_ context.Context) error {
			return context.DeadlineExceeded
		},
	})
	_ = m.Start(context.Background())
	time.Sleep(50 * time.Millisecond)
	select {
	case err := <-m.Err():
		t.Fatalf("unexpected error: %v", err)
	default:
	}
	_ = m.Stop(context.Background())
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("timeout waiting for condition")
}

type qwMockLogger struct {
	mu         sync.Mutex
	infoCount  atomic.Int32
	errorCount atomic.Int32
}

func (m *qwMockLogger) Info(_ string, _ ...any)  { m.infoCount.Add(1) }
func (m *qwMockLogger) Error(_ string, _ ...any) { m.errorCount.Add(1) }
