package commandbus

import (
	"time"

	"github.com/shuldan/commands"
)

// ReceiverOption настраивает CommandReceiver при создании.
type ReceiverOption func(*receiverConfig)

type receiverConfig struct {
	idemStore commands.IdempotencyStore
	idemTTL   time.Duration
}

// WithIdempotencyStore задаёт хранилище идемпотентности.
func WithIdempotencyStore(store commands.IdempotencyStore) ReceiverOption {
	return func(c *receiverConfig) { c.idemStore = store }
}

// WithIdempotencyTTL задаёт глобальный TTL для ключей идемпотентности.
func WithIdempotencyTTL(ttl time.Duration) ReceiverOption {
	return func(c *receiverConfig) { c.idemTTL = ttl }
}

// HandleOption настраивает обработку конкретной команды.
type HandleOption func(*inboundEntry)

// WithCommandIdempotencyTTL переопределяет TTL для конкретной команды.
func WithCommandIdempotencyTTL(ttl time.Duration) HandleOption {
	return func(e *inboundEntry) { e.idemTTL = ttl }
}
