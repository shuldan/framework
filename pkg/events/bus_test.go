package events

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"
)

type TestEvent struct {
	Message string
	Value   int
}

type TestEventOther struct {
	Data string
}

func testListenerFunc(context.Context, TestEvent) error {
	return nil
}

type TestEventListener struct {
	Called bool
	Mutex  sync.Mutex
	Last   TestEvent
}

func (t *TestEventListener) Handle(_ context.Context, e TestEvent) error {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	t.Called = true
	t.Last = e
	return nil
}

func errorListener(context.Context, TestEvent) error {
	return ErrBusClosed
}

func panicListener(context.Context, TestEvent) error {
	panic("test panic")
}

type mockErrorHandler struct {
	Called bool
	Event  any
	Err    error
}

func (m *mockErrorHandler) Handle(event any, _ any, err error) {
	m.Called = true
	m.Event = event
	m.Err = err
}

type mockPanicHandler struct {
	Called     bool
	Event      any
	PanicValue any
}

func (m *mockPanicHandler) Handle(event any, _ any, panicValue any, _ []byte) {
	m.Called = true
	m.Event = event
	m.PanicValue = panicValue
}

func TestEventBus_PublishToFunctionListener(t *testing.T) {
	bus := New()

	err := bus.Subscribe((*TestEvent)(nil), testListenerFunc)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	event := TestEvent{Message: "hello", Value: 100}
	err = bus.Publish(context.Background(), event)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	_ = bus.Close()
}

func TestEventBus_PublishToStructListener(t *testing.T) {
	bus := New()
	listener := &TestEventListener{}

	err := bus.Subscribe((*TestEvent)(nil), listener)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	event := TestEvent{Message: "update", Value: 42}
	err = bus.Publish(context.Background(), event)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	listener.Mutex.Lock()
	if !listener.Called {
		t.Error("listener was not called")
	}
	if listener.Last.Message != "update" || listener.Last.Value != 42 {
		t.Errorf("unexpected event data: %+v", listener.Last)
	}
	listener.Mutex.Unlock()

	_ = bus.Close()
}

func TestEventBus_MultipleListeners(t *testing.T) {
	bus := New()
	listener1 := &TestEventListener{}
	listener2 := &TestEventListener{}

	_ = bus.Subscribe((*TestEvent)(nil), listener1)
	_ = bus.Subscribe((*TestEvent)(nil), listener2)

	event := TestEvent{Message: "broadcast", Value: 1}
	_ = bus.Publish(context.Background(), event)
	time.Sleep(100 * time.Millisecond)

	listener1.Mutex.Lock()
	if !listener1.Called {
		t.Error("listener1 was not called")
	}
	listener1.Mutex.Unlock()

	listener2.Mutex.Lock()
	if !listener2.Called {
		t.Error("listener2 was not called")
	}
	listener2.Mutex.Unlock()

	_ = bus.Close()
}

func TestEventBus_WrongEventType(t *testing.T) {
	bus := New()

	err := bus.Subscribe((*TestEvent)(nil), func(ctx context.Context, e TestEventOther) error {
		return nil
	})

	if err == nil {
		t.Fatal("expected error when event type mismatch")
	}

	if !reflect.DeepEqual(err, ErrInvalidListener.
		WithDetail("expected_type", "events.TestEvent").
		WithDetail("actual_type", "events.TestEventOther")) {
		t.Errorf("unexpected error: %v", err)
	}

	_ = bus.Close()
}

func TestEventBus_InvalidListenerFunction(t *testing.T) {
	bus := New()

	err := bus.Subscribe((*TestEvent)(nil), func(e TestEvent) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error for invalid listener function")
	}

	err = bus.Subscribe((*TestEvent)(nil), func(ctx context.Context, e TestEvent) {

	})
	if err == nil {
		t.Fatal("expected error for invalid return type")
	}

	_ = bus.Close()
}

func TestEventBus_InvalidEventType(t *testing.T) {
	bus := New()

	err := bus.Subscribe("not a pointer", func(ctx context.Context, e TestEvent) error { return nil })
	if err == nil {
		t.Fatal("expected error for invalid eventType")
	}

	err = bus.Subscribe(TestEvent{}, func(ctx context.Context, e TestEvent) error { return nil })
	if err == nil {
		t.Fatal("expected error for non-pointer eventType")
	}

	_ = bus.Close()
}

func TestEventBus_PanicInListener(t *testing.T) {
	panicHandler := &mockPanicHandler{}
	bus := New()
	bus.WithPanicHandler(panicHandler)

	_ = bus.Subscribe((*TestEvent)(nil), panicListener)

	event := TestEvent{Message: "panic"}
	err := bus.Publish(context.Background(), event)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	if !panicHandler.Called {
		t.Errorf("expected panic handler to be called")
	}

	if panicHandler.PanicValue == nil {
		t.Errorf("expected panic value to be set")
	}

	_ = bus.Close()
}

func TestEventBus_ClosedBus(t *testing.T) {
	bus := New()
	_ = bus.Close()

	err := bus.Subscribe((*TestEvent)(nil), func(ctx context.Context, e TestEvent) error {
		return nil
	})
	if !errors.Is(err, ErrBusClosed) {
		t.Errorf("expected ErrBusClosed, got %v", err)
	}

	err = bus.Publish(context.Background(), TestEvent{})
	if !errors.Is(err, ErrPublishOnClosedBus) {
		t.Errorf("expected ErrPublishOnClosedBus, got %v", err)
	}
}

func TestEventBus_ConcurrentPublish(t *testing.T) {
	bus := New()
	listener := &TestEventListener{}

	_ = bus.Subscribe((*TestEvent)(nil), listener)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			event := TestEvent{Message: "conc", Value: val}
			_ = bus.Publish(context.Background(), event)
		}(i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	listener.Mutex.Lock()
	if !listener.Called {
		t.Error("listener was not called")
	}
	listener.Mutex.Unlock()

	_ = bus.Close()
}

func TestEventBus_ErrorHandlerCalled(t *testing.T) {
	mockHandler := &mockErrorHandler{}
	bus := New().WithErrorHandler(mockHandler)

	_ = bus.Subscribe((*TestEvent)(nil), errorListener)
	_ = bus.Publish(context.Background(), TestEvent{})

	time.Sleep(100 * time.Millisecond)

	if !mockHandler.Called {
		t.Error("error handler was not called")
	}
	if mockHandler.Err == nil {
		t.Error("expected error")
	}
}
