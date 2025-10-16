package contracts

import (
	"context"
)

type QueueMessage interface {
	ID() string
	Body() []byte
	Headers() map[string]interface{}
	Ack() error
	Nack(requeue bool) error
}

type QueueProducer interface {
	Publish(ctx context.Context, message []byte, headers map[string]interface{}) error
}

type QueueProcessor interface {
	Process(ctx context.Context, msg QueueMessage) error
}

type QueueConsumer interface {
	Consume(ctx context.Context, processor QueueProcessor) error
}

type QueueManager interface {
	DeclareQueue(name string) error
	DeleteQueue(name string) error
}

type Queue interface {
	QueueProducer
	QueueConsumer
	QueueManager
	Close() error
}
