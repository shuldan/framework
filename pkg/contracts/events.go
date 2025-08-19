package contracts

import (
	"context"
)

type EventPanicHandler interface {
	Handle(event any, listener any, panicValue any, stack []byte)
}

type EventErrorHandler interface {
	Handle(event any, listener any, err error)
}

type Bus interface {
	WithPanicHandler(h EventPanicHandler) Bus
	WithErrorHandler(h EventErrorHandler) Bus
	Subscribe(eventType any, listener any) error
	Publish(ctx context.Context, event any) error
	Close() error
}
