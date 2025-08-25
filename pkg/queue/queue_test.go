package queue

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/shuldan/framework/pkg/contracts"
	"strings"
	"testing"
	"time"
)

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

	err := q.Consume(ctx, func(ctx context.Context, job *TestJob) error {
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
