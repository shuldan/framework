package contracts

import "context"

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
