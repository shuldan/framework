package redis

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestBroker_Produce(t *testing.T) {
	mock := &mockCmdable{}
	b := &broker{
		client: mock,
		config: &config{streamKeyFormat: "stream:%s"},
	}

	mock.expect("XAdd", []interface{}{&redis.XAddArgs{
		Stream: "stream:test",
		Values: map[string]interface{}{"payload": `{"data":"aGVsbG8=","enqueued_at":"` + time.Now().UTC().Format(time.RFC3339) + `"}`},
	}}, "12345-0", nil)

	err := b.Produce(context.Background(), "test", []byte("hello"))
	if err != nil {
		t.Fatalf("Produce failed: %v", err)
	}

	mock.check(t)
}

func TestBroker_Consume_NewGroup(t *testing.T) {
	mock := &mockCmdable{}

	mock.expect("XInfoGroups", []interface{}{"stream:test"}, []redis.XInfoGroup{}, redis.Nil)

	mock.expect("XGroupCreateMkStream", []interface{}{"stream:test", "consumers:test", "0"}, "OK", nil)

	mock.expect("XReadGroup", []interface{}{&redis.XReadGroupArgs{
		Group:    "consumers:test",
		Consumer: "consumer-test-uuid",
		Streams:  []string{"stream:test", ">"},
		Count:    1,
		Block:    500 * time.Millisecond,
	}}, []redis.XStream{}, redis.Nil)

	b := &broker{
		client: mock,
		config: &config{
			streamKeyFormat: "stream:%s",
			consumerGroup:   "consumers",
			claimInterval:   10 * time.Millisecond,
		},
		consumers: make(map[string][]context.CancelFunc),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := b.Consume(ctx, "test", func(data []byte) error {
		return nil
	})
	if err != nil {
		t.Fatalf("Consume failed: %v", err)
	}

	<-ctx.Done()

	mock.mu.Lock()
	defer mock.mu.Unlock()

	var haveXInfo, haveCreate, haveRead bool
	for _, call := range mock.calls {
		switch call.method {
		case "XInfoGroups":
			haveXInfo = true
		case "XGroupCreateMkStream":
			haveCreate = true
		case "XReadGroup":
			haveRead = true
		}
	}

	if !haveXInfo {
		t.Error("XInfoGroups was not called")
	}
	if !haveCreate {
		t.Error("XGroupCreateMkStream was not called")
	}
	if !haveRead {
		t.Error("XReadGroup was not called")
	}
}

func TestBroker_Consume_ExistingGroup(t *testing.T) {
	mock := &mockCmdable{}
	b := &broker{
		client: mock,
		config: &config{
			streamKeyFormat: "stream:%s",
			consumerGroup:   "consumers",
			claimInterval:   10 * time.Millisecond,
		},
		consumers: make(map[string][]context.CancelFunc),
	}

	setupMockExpectationsExistingGroup(mock)

	var received []byte
	var mu sync.Mutex
	done := make(chan struct{})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

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

	verifyMockCallsExistingGroup(t, mock)
}

const xReadGroupMethod = "XReadGroup"

func setupMockExpectationsExistingGroup(mock *mockCmdable) {
	mock.expect("XInfoGroups", []interface{}{"stream:test"}, []redis.XInfoGroup{
		{Name: "consumers:test"},
	}, nil)

	mock.expect(xReadGroupMethod, []interface{}{&redis.XReadGroupArgs{
		Group:    "consumers:test",
		Consumer: "consumer-test-uuid",
		Streams:  []string{"stream:test", ">"},
		Count:    1,
		Block:    500 * time.Millisecond,
	}}, []redis.XStream{{
		Messages: []redis.XMessage{{
			ID: "12345-0",
			Values: map[string]interface{}{
				"payload": `{"data":"aGVsbG8=","enqueued_at":"2025-01-01T00:00:00Z"}`,
			},
		}},
	}}, nil)

	mock.expect("XAck", []interface{}{"stream:test", "consumers:test", []string{"12345-0"}}, int64(1), nil)
}

func verifyMockCallsExistingGroup(t *testing.T, mock *mockCmdable) {
	mock.mu.Lock()
	defer mock.mu.Unlock()

	var haveXInfo, haveRead, haveAck bool
	for _, call := range mock.calls {
		switch call.method {
		case "XInfoGroups":
			haveXInfo = true
		case xReadGroupMethod:
			haveRead = true
		case "XAck":
			haveAck = true
		}
	}

	if !haveXInfo {
		t.Error("XInfoGroups was not called")
	}
	if !haveRead {
		t.Error("XReadGroup was not called")
	}
	if !haveAck {
		t.Error("XAck was not called")
	}
}

func TestBroker_Close(t *testing.T) {
	mock := &mockCmdable{}
	b := &broker{
		client:    mock,
		consumers: make(map[string][]context.CancelFunc),
		config: &config{
			streamKeyFormat: "stream:%s",
			claimInterval:   10 * time.Millisecond,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	consumeDone := make(chan error, 1)
	go func() {
		err := b.Consume(ctx, "test", func(data []byte) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		})
		consumeDone <- err
	}()

	time.Sleep(10 * time.Millisecond)

	err := b.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	select {
	case err := <-consumeDone:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("Consume should return context.Canceled, got: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Consume did not finish in time after Close")
	}
}
