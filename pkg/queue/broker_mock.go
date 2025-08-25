package queue

import (
	"context"
)

type mockBroker struct {
	produceFunc func(ctx context.Context, topic string, data []byte) error
	consumeFunc func(ctx context.Context, topic string, handler func([]byte) error) error
	closeFunc   func() error
}

func (m *mockBroker) Produce(ctx context.Context, topic string, data []byte) error {
	if m.produceFunc != nil {
		return m.produceFunc(ctx, topic, data)
	}
	return nil
}

func (m *mockBroker) Consume(ctx context.Context, topic string, handler func([]byte) error) error {
	if m.consumeFunc != nil {
		return m.consumeFunc(ctx, topic, handler)
	}
	return nil
}

func (m *mockBroker) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}
