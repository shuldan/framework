package broker

import "github.com/shuldan/framework/pkg/errors"

var newErrorCode = errors.WithPrefix("QUEUE_BROKER")

var (
	ErrQueueBrokerConfigNotFound = newErrorCode().New("queue broker config not found")
	ErrRedisConfigNotFound       = newErrorCode().New("redis config not found")
	ErrRedisClientNotConfigured  = newErrorCode().New("redis client not configured")
	ErrInvalidConfigInstance     = newErrorCode().New("config instance must be Config interface")
	ErrInvalidLoggerInstance     = newErrorCode().New("logger instance must be a Logger interface")
	ErrUnsupportedQueueDriver    = newErrorCode().New("unsupported queue driver")
)
