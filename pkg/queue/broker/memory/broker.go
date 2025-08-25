package memory

import (
	"context"
	"github.com/shuldan/framework/pkg/queue"
	"sync"
)

type broker struct {
	mu       sync.RWMutex
	channels map[string]chan []byte
	runners  map[string][]context.CancelFunc
	closed   bool
}

func New() queue.Broker {
	return &broker{
		channels: make(map[string]chan []byte),
		runners:  make(map[string][]context.CancelFunc),
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
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return queue.ErrQueueClosed
	}
	b.mu.RUnlock()

	ch := b.getOrCreateChan(topic)

	ctx, cancel := context.WithCancel(ctx)
	b.mu.Lock()
	b.runners[topic] = append(b.runners[topic], cancel)
	consumerID := len(b.runners[topic]) - 1
	b.mu.Unlock()
	defer func() {
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
	}()

	go func() {
		defer cancel()
		for {
			select {
			case data, ok := <-ch:
				if !ok {
					return
				}
				go func() {
					defer func() {
						if r := recover(); r != nil {
							// log
						}
					}()
					_ = handler(data)
				}()
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
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
