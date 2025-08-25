package redis

import "github.com/shuldan/framework/pkg/errors"

var newRedisBrokerCode = errors.WithPrefix("REDIS_BROKER")

var (
	ErrInvalidPayload     = newRedisBrokerCode().New("missing or invalid 'payload' field")
	ErrProduceFailed      = newRedisBrokerCode().New("failed to produce message to Redis stream")
	ErrEncodeFailed       = newRedisBrokerCode().New("failed to encode message for Redis")
	ErrConsumeSetupFailed = newRedisBrokerCode().New("failed to setup consumer group in Redis")
	ErrGroupCheckFailed   = newRedisBrokerCode().New("failed to check consumer group existence")
)
