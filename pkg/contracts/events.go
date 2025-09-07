package contracts

import (
	"context"
)

type EventBus interface {
	Subscribe(eventType any, listener any) error
	Publish(ctx context.Context, event any) error
	Close() error
}
