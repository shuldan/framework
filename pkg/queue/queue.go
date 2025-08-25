package queue

import (
	"context"
	"encoding/json"
	"reflect"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

type Broker interface {
	Produce(ctx context.Context, topic string, data []byte) error
	Consume(ctx context.Context, topic string, handler func([]byte) error) error
	Close() error
}

type Queue[T any] interface {
	Produce(ctx context.Context, job T) error
	Consume(ctx context.Context, handler func(context.Context, T) error) error
	Close() error
}

type typedQueue[T any] struct {
	topic        string
	broker       Broker
	concurrency  int
	maxRetries   int
	backoff      BackoffStrategy
	errorHandler ErrorHandler
	panicHandler PanicHandler
	mu           sync.RWMutex
	closed       bool
	prefix       string
	dlqEnabled   bool
	counter      Counter
}

func (q *typedQueue[T]) Produce(ctx context.Context, job T) error {
	q.mu.RLock()
	if q.closed {
		q.mu.RUnlock()
		return ErrQueueClosed
	}
	q.mu.RUnlock()

	data, err := json.Marshal(job)
	if err != nil {
		return err
	}

	topic := q.getPrefixedTopic()
	return q.broker.Produce(ctx, topic, data)
}

func (q *typedQueue[T]) Consume(ctx context.Context, handler func(context.Context, T) error) error {
	jobs := make(chan []byte, q.concurrency*10)
	workerCtx, workerCancel := context.WithCancel(ctx)
	defer workerCancel()

	var workerWg sync.WaitGroup
	for i := 0; i < q.concurrency; i++ {
		workerWg.Add(1)
		go func() {
			defer workerWg.Done()
			for {
				select {
				case data, ok := <-jobs:
					if !ok {
						return
					}
					q.processJob(workerCtx, data, handler)
				case <-workerCtx.Done():
					return
				}
			}
		}()
	}

	defer func() {
		close(jobs)
		workerCancel()
		workerWg.Wait()
	}()

	topic := q.getPrefixedTopic()
	if err := q.broker.Consume(ctx, topic, func(data []byte) error {
		select {
		case jobs <- data:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}); err != nil {
		return err
	}

	<-ctx.Done()

	return ctx.Err()
}

func (q *typedQueue[T]) Close() error {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return nil
	}
	q.closed = true
	q.mu.Unlock()

	return q.broker.Close()
}

func (q *typedQueue[T]) processJob(ctx context.Context, data []byte, handler func(context.Context, T) error) {
	startTime := time.Now()
	defer func() {
		if r := recover(); r != nil {
			q.panicHandler.Handle(nil, handler, r, debug.Stack())
		}
	}()

	select {
	case <-ctx.Done():
		return
	default:
	}

	retry := 0
	for {
		var job T
		if err := json.Unmarshal(data, &job); err != nil {
			q.errorHandler.Handle(job, handler, err)
			q.counter.IncError(q.getPrefixedTopic(), handlerName(handler))
			q.counter.IncProcessed(q.getPrefixedTopic(), StatusError)
			return
		}

		err := handler(ctx, job)
		if err == nil {
			q.counter.ObserveProcessingTime(q.getPrefixedTopic(), time.Since(startTime))
			q.counter.IncProcessed(q.getPrefixedTopic(), StatusSuccess)
			return
		}

		q.errorHandler.Handle(job, handler, err)
		q.counter.IncError(q.getPrefixedTopic(), handlerName(handler))

		retry++
		if retry > q.maxRetries {
			if q.dlqEnabled {
				q.sendToDLQ(ctx, job)
				q.counter.IncDLQ(q.getPrefixedTopic())
			}
			q.counter.IncProcessed(q.getPrefixedTopic(), StatusDLQ)
			return
		}

		q.counter.IncRetry(q.getPrefixedTopic())
		delay := q.backoff.Delay(retry)
		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-timer.C:
			case <-ctx.Done():
				if !timer.Stop() {
					<-timer.C
				}
				return
			}
		}
	}
}

func (q *typedQueue[T]) sendToDLQ(ctx context.Context, job T) {
	data, err := json.Marshal(job)
	if err != nil {
		q.errorHandler.Handle(job, nil, ErrMarshal.WithCause(err))
		return
	}

	dlqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	topic := q.getDLQTopic()
	if err := q.broker.Produce(dlqCtx, topic, data); err != nil {
		q.errorHandler.Handle(job, nil, ErrSendToDLQ.WithCause(err))
	}
}

func (q *typedQueue[T]) getPrefixedTopic() string {
	if q.prefix == "" {
		return q.topic
	}
	return q.prefix + q.topic
}

func (q *typedQueue[T]) getDLQTopic() string {
	dlqTopic := "dlq:" + q.topic
	if q.prefix == "" {
		return dlqTopic
	}
	return q.prefix + dlqTopic
}

func handlerName(handler interface{}) string {
	if handler == nil {
		return "unknown"
	}
	return runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()
}
