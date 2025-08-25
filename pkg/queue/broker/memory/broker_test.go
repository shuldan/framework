package memory

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestBroker_ProduceConsume(t *testing.T) {
	b := New()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var received []byte
	err := b.Consume(ctx, "test", func(data []byte) error {
		received = data
		return nil
	})
	if err != nil {
		t.Fatalf("Consume failed: %v", err)
	}

	err = b.Produce(context.Background(), "test", []byte("hello"))
	if err != nil {
		t.Fatalf("Produce failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	if string(received) != "hello" {
		t.Errorf("expected 'hello', got %q", string(received))
	}
}

func TestBroker_ConcurrentConsume(t *testing.T) {
	b := New()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var count int
	var mu sync.Mutex
	done := make(chan struct{}, 1)
	expected := 5

	for i := 0; i < 3; i++ {
		err := b.Consume(ctx, "shared", func([]byte) error {
			mu.Lock()
			count++
			if count >= expected {
				select {
				case done <- struct{}{}:
				default:
				}
			}
			mu.Unlock()
			return nil
		})
		if err != nil {
			t.Fatalf("Consume failed: %v", err)
		}
	}

	for i := 0; i < expected; i++ {
		_ = b.Produce(context.Background(), "shared", []byte("msg"))
	}

	select {
	case <-done:

	case <-ctx.Done():
		t.Fatalf("Timeout: expected %d messages, got %d", expected, count)
	}
}

func TestBroker_Close(t *testing.T) {
	b := New()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := b.Consume(ctx, "test", func([]byte) error { return nil })
	if err != nil {
		t.Fatal(err)
	}

	err = b.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	err = b.Produce(context.Background(), "test", []byte("after"))
	if err == nil {
		t.Error("expected error after close, got nil")
	}

	err = b.Consume(context.Background(), "test", func([]byte) error { return nil })
	if err == nil {
		t.Error("expected error on Consume after Close")
	}
}
