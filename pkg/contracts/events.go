package contracts

import (
	"context"
)

type Bus interface {
	Subscribe(eventType any, listener any) error
	Publish(ctx context.Context, event any) error
	Close() error
}
