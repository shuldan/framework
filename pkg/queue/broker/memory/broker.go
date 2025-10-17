package memory

import (
	"context"
	"log/slog"
	"runtime/debug"
	"sync"

	"github.com/shuldan/framework/pkg/contracts"

	"github.com/shuldan/framework/pkg/queue"
)

type broker struct {
	mu       sync.RWMutex
	channels map[string]chan []byte
	runners  map[string][]context.CancelFunc
	closed   bool
	logger   contracts.Logger
}

func New(logger contracts.Logger) contracts.Broker {
	return &broker{
		channels: make(map[string]chan []byte),
		runners:  make(map[string][]context.CancelFunc),
		logger:   logger,
	}
}

func (b *broker) getOrCreateChan(topic string) chan []byte {
	b.mu.Lock()
	defer b.mu.Unlock()

	if ch, ok := b.channels[topic]; ok {
		return ch
	}

	ch := make(chan []byte, 100)
	b.channels[topic] = ch
	return ch
}

func (b *broker) Produce(ctx context.Context, topic string, data []byte) error {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return context.Canceled
	}
	b.mu.RUnlock()

	ch := b.getOrCreateChan(topic)
	select {
	case ch <- data:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (b *broker) Consume(ctx context.Context, topic string, handler func([]byte) error) error {
	if err := b.validateConsume(ctx); err != nil {
		return err
	}

	ch := b.getOrCreateChan(topic)
	ctx, cancel := context.WithCancel(ctx)
	consumerID := b.registerConsumer(topic, cancel)
	defer b.unregisterConsumer(topic, consumerID)

	go b.consumeMessages(ctx, ch, topic, handler)
	return nil
}

func (b *broker) validateConsume(ctx context.Context) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return queue.ErrQueueClosed
	}
	return nil
}

func (b *broker) registerConsumer(topic string, cancel context.CancelFunc) int {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.runners[topic] = append(b.runners[topic], cancel)
	return len(b.runners[topic]) - 1
}

func (b *broker) unregisterConsumer(topic string, consumerID int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	cancels, exists := b.runners[topic]
	if !exists || consumerID >= len(cancels) || cancels[consumerID] == nil {
		return
	}

	lastIndex := len(cancels) - 1
	if consumerID != lastIndex {
		cancels[consumerID] = cancels[lastIndex]
	}
	b.runners[topic] = cancels[:lastIndex]
	cancels[lastIndex] = nil
}

func (b *broker) consumeMessages(ctx context.Context, ch chan []byte, topic string, handler func([]byte) error) {
	defer func() {
		if r := recover(); r != nil {
			b.handlePanic(topic, r)
		}
	}()

	for {
		select {
		case data, ok := <-ch:
			if !ok {
				return
			}
			b.handleMessage(data, topic, handler)
		case <-ctx.Done():
			return
		}
	}
}

func (b *broker) handleMessage(data []byte, topic string, handler func([]byte) error) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				b.handlePanic(topic, r)
			}
		}()
		_ = handler(data)
	}()
}

func (b *broker) handlePanic(topic string, r interface{}) {
	if b.logger != nil {
		b.logger.Error("panic in message handler",
			"topic", topic,
			"panic", r,
			"stack", string(debug.Stack()))
		return
	}
	slog.Error(
		"panic in message handler",
		"topic", topic,
		"panic", r,
		"stack", string(debug.Stack()),
	)
}

func (b *broker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	for topic, cancels := range b.runners {
		for _, cancel := range cancels {
			if cancel != nil {
				cancel()
			}
		}
		b.runners[topic] = nil
	}

	for topic := range b.channels {
		close(b.channels[topic])
		delete(b.channels, topic)
	}

	b.closed = true
	return nil
}
