package contracts

import (
	"context"
)

type IQueueMessage interface {
	ID() string
	Body() []byte
	Headers() map[string]interface{}
	Ack() error
	Nack(requeue bool) error
}

type IQueueProducer interface {
	Publish(ctx context.Context, message []byte, headers map[string]interface{}) error
}

type IQueueProcessor interface {
	Process(ctx context.Context, msg IQueueMessage) error
}

type IQueueConsumer interface {
	Consume(ctx context.Context, processor IQueueProcessor) error
}

type IQueueManager interface {
	DeclareQueue(name string) error
	DeleteQueue(name string) error
}

type IQueue interface {
	IQueueProducer
	IQueueConsumer
	IQueueManager
	Close() error
}
