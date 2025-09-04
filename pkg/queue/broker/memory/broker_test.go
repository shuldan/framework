package memory

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/shuldan/framework/pkg/errors"
	"github.com/shuldan/framework/pkg/queue"
)

func TestBroker_ProduceConsume(t *testing.T) {
	b := New(nil)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var received []byte
	var mu sync.Mutex
	done := make(chan struct{})

	err := b.Consume(ctx, "test", func(data []byte) error {
		mu.Lock()
		received = data
		mu.Unlock()
		close(done)
		return nil
	})
	if err != nil {
		t.Fatalf("Consume failed: %v", err)
	}

	err = b.Produce(context.Background(), "test", []byte("hello"))
	if err != nil {
		t.Fatalf("Produce failed: %v", err)
	}

	select {
	case <-done:

	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for message")
	}

	mu.Lock()
	receivedStr := string(received)
	mu.Unlock()

	if receivedStr != "hello" {
		t.Errorf("expected 'hello', got %q", receivedStr)
	}
}

func TestBroker_ConcurrentConsume(t *testing.T) {
	b := New(nil)
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
		mu.Lock()
		actualCount := count
		mu.Unlock()
		t.Fatalf("Timeout: expected %d messages, got %d", expected, actualCount)
	}
}

func TestBroker_Close(t *testing.T) {
	b := New(nil)

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

func TestBroker_ProduceCanceledContext(t *testing.T) {
	t.Parallel()
	b := New(nil)
	ctx, cancel := context.WithCancel(context.Background())

	for i := 0; i < 100; i++ {
		err := b.Produce(context.Background(), "test", []byte("fill"))
		if err != nil {
			t.Fatalf("failed to fill buffer: %v", err)
		}
	}

	cancel()

	err := b.Produce(ctx, "test", []byte("should fail"))
	if err == nil {
		t.Error("expected error for canceled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestBroker_HandlerPanic(t *testing.T) {
	b := New(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	panicChan := make(chan struct{})
	err := b.Consume(ctx, "panic-test", func(data []byte) error {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic to be recovered")
			}
			close(panicChan)
		}()
		panic("test panic")
	})
	if err != nil {
		t.Fatalf("Consume failed: %v", err)
	}

	err = b.Produce(context.Background(), "panic-test", []byte("trigger"))
	if err != nil {
		t.Fatalf("Produce failed: %v", err)
	}

	select {
	case <-panicChan:
	case <-time.After(200 * time.Millisecond):
		t.Error("panic was not handled")
	}
}

func TestBroker_ConsumeAfterClose(t *testing.T) {
	t.Parallel()
	b := New(nil)

	err := b.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	err = b.Consume(context.Background(), "test", func([]byte) error { return nil })
	if !errors.Is(err, queue.ErrQueueClosed) {
		t.Errorf("expected ErrQueueClosed, got %v", err)
	}
}

func TestBroker_MultipleTopic(t *testing.T) {
	t.Parallel()
	b := New(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	var receivedTopic1, receivedTopic2 []byte
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(2)
	err := b.Consume(ctx, "topic1", func(data []byte) error {
		mu.Lock()
		receivedTopic1 = data
		mu.Unlock()
		wg.Done()
		return nil
	})
	if err != nil {
		t.Fatalf("Consume topic1 failed: %v", err)
	}
	err = b.Consume(ctx, "topic2", func(data []byte) error {
		mu.Lock()
		receivedTopic2 = data
		mu.Unlock()
		wg.Done()
		return nil
	})
	if err != nil {
		t.Fatalf("Consume topic2 failed: %v", err)
	}
	err = b.Produce(context.Background(), "topic1", []byte("msg1"))
	if err != nil {
		t.Fatalf("Produce topic1 failed: %v", err)
	}
	err = b.Produce(context.Background(), "topic2", []byte("msg2"))
	if err != nil {
		t.Fatalf("Produce topic2 failed: %v", err)
	}
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for messages")
	}
	mu.Lock()
	r1, r2 := string(receivedTopic1), string(receivedTopic2)
	mu.Unlock()
	if r1 != "msg1" {
		t.Errorf("expected 'msg1' on topic1, got %q", r1)
	}
	if r2 != "msg2" {
		t.Errorf("expected 'msg2' on topic2, got %q", r2)
	}
}
