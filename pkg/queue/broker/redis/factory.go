package redis

import (
	"context"

	"github.com/redis/go-redis/v9"

	"github.com/shuldan/framework/pkg/queue"
)

func New(client *redis.Client, opts ...Option) queue.Broker {
	c := defaultConfig()
	for _, opt := range opts {
		opt(c)
	}

	return &broker{
		client:    client,
		consumers: make(map[string][]context.CancelFunc),
		config:    c,
	}
}
