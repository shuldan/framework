package queue

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/shuldan/framework/pkg/contracts"
)

type TestJob struct {
	Data string
}

type noOpLogger struct{}

func (l *noOpLogger) Debug(msg string, fields ...interface{})    {}
func (l *noOpLogger) Info(msg string, fields ...interface{})     {}
func (l *noOpLogger) Warn(msg string, fields ...interface{})     {}
func (l *noOpLogger) Error(msg string, fields ...interface{})    {}
func (l *noOpLogger) Critical(msg string, fields ...interface{}) {}
func (l *noOpLogger) Trace(msg string, args ...any)              {}
func (l *noOpLogger) With(args ...any) contracts.Logger          { return l }

func TestQueue_Produce_Closed(t *testing.T) {
	broker := &mockBroker{}
	q, _ := New[*TestJob](broker)

	_ = q.Close()

	err := q.Produce(context.Background(), &TestJob{Data: "test"})
	if err == nil {
		t.Fatal("expected error on produce to closed queue")
	}
	if !errors.Is(err, ErrQueueClosed) {
		t.Errorf("expected ErrQueueClosed, got %v", err)
	}
}

func TestQueue_Consume_ProcessJob_Success(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var handled *TestJob
	var handlerCtx context.Context

	broker := &mockBroker{
		consumeFunc: func(brokerCtx context.Context, topic string, handler func([]byte) error) error {
			if err := handler([]byte(`{"Data":"success"}`)); err != nil {
				t.Logf("Delivery error: %v", err)
			}
			return nil
		},
	}

	q, _ := New[*TestJob](broker, WithConcurrency(1))

	err := q.Consume(ctx, func(c context.Context, job *TestJob) error {
		handlerCtx = c
		handled = job
		cancel()
		return nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	if handled == nil {
		t.Fatal("handler was not called")
	}
	if handled.Data != "success" {
		t.Errorf("expected 'success', got %q", handled.Data)
	}
	if !errors.Is(handlerCtx.Err(), context.Canceled) {
		t.Error("handler should receive canceled context")
	}
}

func TestQueue_Consume_RetryWithBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	broker := &mockBroker{
		consumeFunc: func(ctx context.Context, topic string, handler func([]byte) error) error {
			_ = handler([]byte(`{"Data":"retry"}`))
			_ = handler([]byte(`{"Data":"retry"}`))
			_ = handler([]byte(`{"Data":"retry"}`))
			return nil
		},
	}

	var retryCount int
	logger := &noOpLogger{}
	q, _ := New[*TestJob](broker,
		WithMaxRetries(2),
		WithBackoff(FixedBackoff{Duration: 1 * time.Millisecond}),
		WithErrorHandler(NewDefaultErrorHandler(logger)),
		WithPanicHandler(NewDefaultPanicHandler(logger)),
	)

	err := q.Consume(ctx, func(_ context.Context, job *TestJob) error {
		retryCount++
		if retryCount < 3 {
			return errors.New("transient error")
		}
		if retryCount >= 3 {
			cancel()
		}
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	if retryCount != 3 {
		t.Errorf("expected 3 retries, got %d", retryCount)
	}
}

func TestQueue_Consume_DeliverToDLQ(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var dlqData []byte
	broker := &mockBroker{
		consumeFunc: func(ctx context.Context, topic string, handler func([]byte) error) error {
			go func() {
				time.Sleep(100 * time.Millisecond)

				_ = handler([]byte(`{"Data":"dlq"}`))
				time.Sleep(200 * time.Millisecond)
				cancel()
			}()
			return nil
		},
		produceFunc: func(ctx context.Context, topic string, data []byte) error {
			if strings.HasPrefix(topic, "dlq:") {
				dlqData = data
			}
			return nil
		},
	}

	logger := &noOpLogger{}
	q, _ := New[*TestJob](broker,
		WithMaxRetries(1),
		WithDLQ(true),
		WithErrorHandler(NewDefaultErrorHandler(logger)),
		WithPanicHandler(NewDefaultPanicHandler(logger)),
	)

	_ = q.Consume(ctx, func(ctx context.Context, job *TestJob) error {
		return errors.New("permanent error")
	})

	if dlqData == nil {
		t.Fatal("expected message sent to DLQ")
	}

	var job TestJob
	_ = json.Unmarshal(dlqData, &job)
	if job.Data != "dlq" {
		t.Errorf("expected DLQ job with Data='dlq', got %q", job.Data)
	}
}

func TestQueue_ProcessJob_InvalidJSON(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	broker := &mockBroker{
		consumeFunc: func(ctx context.Context, topic string, handler func([]byte) error) error {
			_ = handler([]byte(`invalid json`))
			return nil
		},
	}

	q, _ := New[*TestJob](broker)
	_ = q.Consume(ctx, func(context.Context, *TestJob) error {
		t.Error("handler should not be called for invalid JSON")
		return nil
	})

	<-ctx.Done()
}

func TestQueue_ProcessJob_Panic(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	broker := &mockBroker{
		consumeFunc: func(ctx context.Context, topic string, handler func([]byte) error) error {
			_ = handler([]byte(`{"Data":"panic"}`))
			return nil
		},
	}

	q, _ := New[*TestJob](broker)
	_ = q.Consume(ctx, func(context.Context, *TestJob) error {
		panic("test panic")
	})

	<-ctx.Done()
}

func TestQueue_ProcessJob_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	broker := &mockBroker{
		consumeFunc: func(ctx context.Context, topic string, handler func([]byte) error) error {
			_ = handler([]byte(`{"Data":"test"}`))
			return nil
		},
	}

	q, _ := New[*TestJob](broker)
	_ = q.Consume(ctx, func(context.Context, *TestJob) error {
		t.Error("handler should not be called for canceled context")
		return nil
	})
}

func TestQueue_GetPrefixedTopic(t *testing.T) {
	t.Parallel()
	broker := &mockBroker{}

	q1, _ := New[*TestJob](broker)
	tq1 := q1.(*typedQueue[*TestJob])
	if topic := tq1.getPrefixedTopic(); !strings.Contains(topic, "TestJob") {
		t.Errorf("expected topic to contain 'TestJob', got %q", topic)
	}

	q2, _ := New[*TestJob](broker, WithPrefix("test-"))
	tq2 := q2.(*typedQueue[*TestJob])
	topic := tq2.getPrefixedTopic()
	if !strings.HasPrefix(topic, "test-") {
		t.Errorf("expected topic to start with 'test-', got %q", topic)
	}
}

func TestQueue_GetDLQTopic(t *testing.T) {
	t.Parallel()
	broker := &mockBroker{}

	q1, _ := New[*TestJob](broker)
	tq1 := q1.(*typedQueue[*TestJob])
	dlqTopic := tq1.getDLQTopic()
	if !strings.HasPrefix(dlqTopic, "dlq:") {
		t.Errorf("expected DLQ topic to start with 'dlq:', got %q", dlqTopic)
	}

	q2, _ := New[*TestJob](broker, WithPrefix("test-"))
	tq2 := q2.(*typedQueue[*TestJob])
	dlqTopic = tq2.getDLQTopic()
	if !strings.HasPrefix(dlqTopic, "test-dlq:") {
		t.Errorf("expected prefixed DLQ topic to start with 'test-dlq:', got %q", dlqTopic)
	}
}

func TestQueue_SendToDLQ_MarshalError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	type BadJob struct {
		Channel chan int `json:"-"`
	}

	broker := &mockBroker{
		consumeFunc: func(ctx context.Context, topic string, handler func([]byte) error) error {
			_ = handler([]byte(`{"Channel":null}`))
			return nil
		},
	}

	q, _ := New[*BadJob](broker, WithDLQ(true), WithMaxRetries(0))
	_ = q.Consume(ctx, func(context.Context, *BadJob) error {
		return errors.New("error")
	})

	<-ctx.Done()
}

func TestQueue_SendToDLQ_ProduceError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	broker := &mockBroker{
		consumeFunc: func(ctx context.Context, topic string, handler func([]byte) error) error {
			_ = handler([]byte(`{"Data":"test"}`))
			return nil
		},
		produceFunc: func(ctx context.Context, topic string, data []byte) error {
			if strings.HasPrefix(topic, "dlq:") {
				return errors.New("DLQ produce error")
			}
			return nil
		},
	}

	q, _ := New[*TestJob](broker, WithDLQ(true), WithMaxRetries(0))
	_ = q.Consume(ctx, func(context.Context, *TestJob) error {
		return errors.New("error")
	})

	<-ctx.Done()
}

func TestQueue_CloseMultipleTimes(t *testing.T) {
	t.Parallel()
	broker := &mockBroker{}
	q, _ := New[*TestJob](broker)

	err1 := q.Close()
	if err1 != nil {
		t.Errorf("first Close() failed: %v", err1)
	}

	err2 := q.Close()
	if err2 != nil {
		t.Errorf("second Close() failed: %v", err2)
	}
}

func TestHandlerName(t *testing.T) {
	t.Parallel()

	name1 := handlerName(nil)
	if name1 != "unknown" {
		t.Errorf("expected 'unknown' for nil handler, got %q", name1)
	}

	testHandler := func() {}
	name2 := handlerName(testHandler)
	if name2 == "" {
		t.Error("expected non-empty name for valid handler")
	}
}

func TestQueue_ConcurrentWorkers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	var processedCount int32
	var mu sync.Mutex
	broker := &mockBroker{
		consumeFunc: func(ctx context.Context, topic string, handler func([]byte) error) error {
			for i := 0; i < 10; i++ {
				_ = handler([]byte(`{"Data":"test"}`))
			}
			return nil
		},
	}

	q, _ := New[*TestJob](broker, WithConcurrency(3))
	_ = q.Consume(ctx, func(context.Context, *TestJob) error {
		mu.Lock()
		processedCount++
		mu.Unlock()
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	<-ctx.Done()

	mu.Lock()
	count := processedCount
	mu.Unlock()

	if count == 0 {
		t.Error("expected some messages to be processed")
	}
}
