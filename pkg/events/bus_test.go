package events

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/shuldan/framework/pkg/contracts"
)

type noOpLogger struct{}

func (l *noOpLogger) Debug(_ string, _ ...interface{})    {}
func (l *noOpLogger) Info(_ string, _ ...interface{})     {}
func (l *noOpLogger) Warn(_ string, _ ...interface{})     {}
func (l *noOpLogger) Error(_ string, _ ...interface{})    {}
func (l *noOpLogger) Critical(_ string, _ ...interface{}) {}
func (l *noOpLogger) Trace(_ string, _ ...any)            {}
func (l *noOpLogger) With(_ ...any) contracts.Logger      { return l }

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
	b := New()

	err := b.Subscribe((*TestEvent)(nil), testListenerFunc)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	event := TestEvent{Message: "hello", Value: 100}
	err = b.Publish(context.Background(), event)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	_ = b.Close()
}

func TestEventBus_PublishToStructListener(t *testing.T) {
	b := New()
	listener := &TestEventListener{}

	err := b.Subscribe((*TestEvent)(nil), listener)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	event := TestEvent{Message: "update", Value: 42}
	err = b.Publish(context.Background(), event)
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

	_ = b.Close()
}

func TestEventBus_MultipleListeners(t *testing.T) {
	b := New()
	listener1 := &TestEventListener{}
	listener2 := &TestEventListener{}

	_ = b.Subscribe((*TestEvent)(nil), listener1)
	_ = b.Subscribe((*TestEvent)(nil), listener2)

	event := TestEvent{Message: "broadcast", Value: 1}
	_ = b.Publish(context.Background(), event)
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

	_ = b.Close()
}

func TestEventBus_WrongEventType(t *testing.T) {
	b := New()

	err := b.Subscribe((*TestEvent)(nil), func(ctx context.Context, e TestEventOther) error {
		return nil
	})

	if err == nil {
		t.Fatal("expected error when event type mismatch")
	}

	expected := ErrInvalidListener.
		WithDetail("expected_type", "events.TestEvent").
		WithDetail("actual_type", "events.TestEventOther")

	if !errors.Is(err, expected) {
		t.Errorf("unexpected error: %v", err)
	}

	_ = b.Close()
}

func TestEventBus_InvalidListenerFunction(t *testing.T) {
	b := New()

	err := b.Subscribe((*TestEvent)(nil), func(e TestEvent) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error for invalid listener function")
	}

	err = b.Subscribe((*TestEvent)(nil), func(ctx context.Context, e TestEvent) {

	})
	if err == nil {
		t.Fatal("expected error for invalid return type")
	}

	_ = b.Close()
}

func TestEventBus_InvalidEventType(t *testing.T) {
	b := New()

	err := b.Subscribe("not a pointer", func(ctx context.Context, e TestEvent) error { return nil })
	if err == nil {
		t.Fatal("expected error for invalid eventType")
	}

	err = b.Subscribe(TestEvent{}, func(ctx context.Context, e TestEvent) error { return nil })
	if err == nil {
		t.Fatal("expected error for non-pointer eventType")
	}

	_ = b.Close()
}

func TestEventBus_PanicInListener(t *testing.T) {
	panicHandler := &mockPanicHandler{}
	b := New(
		WithPanicHandler(panicHandler),
	)

	_ = b.Subscribe((*TestEvent)(nil), panicListener)

	event := TestEvent{Message: "panic"}
	err := b.Publish(context.Background(), event)
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

	_ = b.Close()
}

func TestEventBus_ClosedBus(t *testing.T) {
	b := New()
	_ = b.Close()

	err := b.Subscribe((*TestEvent)(nil), func(ctx context.Context, e TestEvent) error {
		return nil
	})
	if !errors.Is(err, ErrBusClosed) {
		t.Errorf("expected ErrBusClosed, got %v", err)
	}

	err = b.Publish(context.Background(), TestEvent{})
	if !errors.Is(err, ErrPublishOnClosedBus) {
		t.Errorf("expected ErrPublishOnClosedBus, got %v", err)
	}
}

func TestEventBus_ConcurrentPublish(t *testing.T) {
	b := New()
	listener := &TestEventListener{}

	_ = b.Subscribe((*TestEvent)(nil), listener)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			event := TestEvent{Message: "conc", Value: val}
			_ = b.Publish(context.Background(), event)
		}(i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	listener.Mutex.Lock()
	if !listener.Called {
		t.Error("listener was not called")
	}
	listener.Mutex.Unlock()

	_ = b.Close()
}

func TestEventBus_ErrorHandlerCalled(t *testing.T) {
	mockHandler := &mockErrorHandler{}
	b := New(
		WithErrorHandler(mockHandler),
	)

	_ = b.Subscribe((*TestEvent)(nil), errorListener)
	_ = b.Publish(context.Background(), TestEvent{})

	time.Sleep(100 * time.Millisecond)

	if !mockHandler.Called {
		t.Error("error handler was not called")
	}
	if mockHandler.Err == nil {
		t.Error("expected error")
	}
}

func TestEventBus_NilEvent(t *testing.T) {
	b := New()
	err := b.Publish(context.Background(), nil)
	if err != nil {
		t.Errorf("publishing nil event should not error: %v", err)
	}
}

func TestEventBus_ContextCancellation(t *testing.T) {
	logger := &noOpLogger{}
	b := New(
		WithErrorHandler(NewDefaultErrorHandler(logger)),
		WithPanicHandler(NewDefaultPanicHandler(logger)),
	)
	ctx, cancel := context.WithCancel(context.Background())

	listener := func(ctx context.Context, event TestEvent) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			return nil
		}
	}

	err := b.Subscribe((*TestEvent)(nil), listener)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	cancel()

	err = b.Publish(ctx, TestEvent{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestListenerAdapter_HandleEvent_InvalidType(t *testing.T) {
	t.Parallel()

	adapter := &listenerAdapter{
		eventType: reflect.TypeOf(TestEvent{}),
		listenerFunc: func(ctx context.Context, event any) error {
			return nil
		},
	}

	err := adapter.handleEvent(context.Background(), TestEventOther{})
	if err == nil {
		t.Fatal("expected error for invalid event type")
	}

	if !errors.Is(err, ErrInvalidEventType) {
		t.Errorf("expected ErrInvalidEventType, got %v", err)
	}
}

func TestAdapterFromFunction_InvalidSignatures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fn   any
	}{
		{"no args", func() error { return nil }},
		{"one arg", func(ctx context.Context) error { return nil }},
		{"three args", func(ctx context.Context, e TestEvent, x int) error { return nil }},
		{"no return", func(ctx context.Context, e TestEvent) {}},
		{"two returns", func(ctx context.Context, e TestEvent) (int, error) { return 0, nil }},
		{"wrong first arg", func(s string, e TestEvent) error { return nil }},
		{"wrong return type", func(ctx context.Context, e TestEvent) string { return "" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := adapterFromFunction(tt.fn)
			if err == nil {
				t.Errorf("expected error for %s", tt.name)
			}
		})
	}
}

func TestAdapterFromMethod_InvalidSignatures(t *testing.T) {
	t.Parallel()

	type InvalidListener struct{}

	var listener InvalidListener
	listenerVal := reflect.ValueOf(listener)

	tests := []struct {
		name   string
		method reflect.Method
	}{
		{
			"wrong arg count",
			reflect.Method{
				Type: reflect.TypeOf(func(InvalidListener, context.Context) error { return nil }),
				Func: reflect.ValueOf(func(InvalidListener, context.Context) error { return nil }),
			},
		},
		{
			"wrong first arg",
			reflect.Method{
				Type: reflect.TypeOf(func(InvalidListener, string, TestEvent) error { return nil }),
				Func: reflect.ValueOf(func(InvalidListener, string, TestEvent) error { return nil }),
			},
		},
		{
			"no return",
			reflect.Method{
				Type: reflect.TypeOf(func(InvalidListener, context.Context, TestEvent) {}),
				Func: reflect.ValueOf(func(InvalidListener, context.Context, TestEvent) {}),
			},
		},
		{
			"wrong return type",
			reflect.Method{
				Type: reflect.TypeOf(func(InvalidListener, context.Context, TestEvent) string { return "" }),
				Func: reflect.ValueOf(func(InvalidListener, context.Context, TestEvent) string { return "" }),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := adapterFromMethod(listenerVal, tt.method)
			if err == nil {
				t.Errorf("expected error for %s", tt.name)
			}
		})
	}
}

func TestNewListenerAdapter_InvalidListener(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		listener any
	}{
		{"nil", nil},
		{"invalid value", (*int)(nil)},
		{"no handle method", struct{}{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newListenerAdapter(tt.listener)
			if err == nil {
				t.Errorf("expected error for %s", tt.name)
			}
		})
	}
}

type TestAsyncListener struct {
	mu     sync.Mutex
	events []TestEvent
}

func (t *TestAsyncListener) Handle(_ context.Context, e TestEvent) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = append(t.events, e)
	return nil
}

func TestEventBus_AsyncMode(t *testing.T) {
	t.Parallel()

	b := New(
		WithAsyncMode(true),
		WithWorkerCount(2),
	)
	defer func() {
		if err := b.Close(); err != nil {
			t.Errorf("Close failed: %v", err)
		}
	}()

	listener := &TestAsyncListener{}
	err := b.Subscribe((*TestEvent)(nil), listener)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	for i := 0; i < 5; i++ {
		event := TestEvent{Message: "async", Value: i}
		err = b.Publish(context.Background(), event)
		if err != nil {
			t.Fatalf("Publish failed: %v", err)
		}
	}

	time.Sleep(200 * time.Millisecond)

	listener.mu.Lock()
	count := len(listener.events)
	listener.mu.Unlock()

	if count != 5 {
		t.Errorf("expected 5 events, got %d", count)
	}
}

func TestEventBus_AsyncContextCancellation(t *testing.T) {
	t.Parallel()

	b := New(
		WithAsyncMode(true),
		WithWorkerCount(1),
	)
	defer func() {
		_ = b.Close()
	}()

	_ = b.Subscribe((*TestEvent)(nil), func(_ context.Context, _ TestEvent) error {
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := b.Publish(ctx, TestEvent{})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func setupBlockedChannelBus() (*eventBus, *sync.Mutex, func() (context.Context, context.CancelFunc)) {
	b := New(
		WithAsyncMode(true),
		WithWorkerCount(1),
	)

	var blockingMu sync.Mutex
	blockingMu.Lock()

	slowListener := func(_ context.Context, _ TestEvent) error {
		blockingMu.Lock()
		defer blockingMu.Unlock()
		return nil
	}

	_ = b.Subscribe((*TestEvent)(nil), slowListener)

	ctxFunc := func() (context.Context, context.CancelFunc) {
		return context.WithTimeout(context.Background(), 50*time.Millisecond)
	}

	return b.(*eventBus), &blockingMu, ctxFunc
}

func publishEventsAndCollectErrors(b *eventBus, ctx context.Context, count int) <-chan error {
	publishErrors := make(chan error, count)
	var publishWg sync.WaitGroup

	for i := 0; i < count; i++ {
		publishWg.Add(1)
		go func(val int) {
			defer publishWg.Done()
			if err := b.Publish(ctx, TestEvent{Value: val}); err != nil {
				select {
				case publishErrors <- err:
				default:
				}
			}
		}(i)
	}

	go func() {
		publishWg.Wait()
		close(publishErrors)
	}()

	return publishErrors
}

func TestEventBus_AsyncChannelBlocked(t *testing.T) {
	t.Parallel()
	busImpl, blockingMu, ctxFunc := setupBlockedChannelBus()
	ctx, cancel := ctxFunc()
	defer cancel()
	channelCapacity := cap(busImpl.eventChan)
	publishErrors := publishEventsAndCollectErrors(busImpl, ctx, channelCapacity+5)
	time.Sleep(25 * time.Millisecond)
	finalErr := busImpl.Publish(ctx, TestEvent{Message: "final"})
	if finalErr == nil {
		t.Errorf("Expected publish to fail due to context deadline, but got nil")
	} else if !errors.Is(finalErr, context.DeadlineExceeded) {
		t.Errorf("Expected context.DeadlineExceeded, got %v", finalErr)
	}
	blockingMu.Unlock()
	time.Sleep(10 * time.Millisecond)
	var errCount int
	for err := range publishErrors {
		if err != nil {
			errCount++
		}
	}
	if errCount == 0 {
		t.Error("Expected some publish errors due to context timeout")
	}
}

func TestEventBus_SubscribeNilEventType(t *testing.T) {
	t.Parallel()

	b := New()
	defer func() {
		_ = b.Close()
	}()

	err := b.Subscribe(nil, func(_ context.Context, _ TestEvent) error {
		return nil
	})

	if err == nil {
		t.Fatal("expected error for nil event type")
	}

	if !errors.Is(err, ErrInvalidEventType) {
		t.Errorf("expected ErrInvalidEventType, got %v", err)
	}
}

func TestEventBus_PublishUnknownEventType(t *testing.T) {
	t.Parallel()

	b := New()
	defer func() {
		_ = b.Close()
	}()

	err := b.Publish(context.Background(), TestEventOther{Data: "unknown"})
	if err != nil {
		t.Errorf("publishing unknown event type should not error: %v", err)
	}
}

func TestEventBus_CloseIdempotent(t *testing.T) {
	t.Parallel()

	b := New()

	err := b.Close()
	if err != nil {
		t.Errorf("first Close failed: %v", err)
	}

	err = b.Close()
	if err != nil {
		t.Errorf("second Close failed: %v", err)
	}
}
