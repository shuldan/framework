package redis

import "time"

type Option func(*config)

type config struct {
	streamKeyFormat   string
	consumerGroup     string
	processingTimeout time.Duration
	claimInterval     time.Duration
	maxClaimBatch     int
	blockTimeout      time.Duration
	maxStreamLength   int64
	approximateTrim   bool
	enableClaim       bool
	consumerPrefix    string
}

func defaultConfig() *config {
	return &config{
		streamKeyFormat:   "stream:%s",
		consumerGroup:     "consumers",
		processingTimeout: 30 * time.Second,
		claimInterval:     1 * time.Second,
		maxClaimBatch:     10,
		blockTimeout:      500 * time.Millisecond,
		maxStreamLength:   0,
		approximateTrim:   true,
		enableClaim:       true,
		consumerPrefix:    "",
	}
}

func WithStreamKeyFormat(format string) Option {
	return func(c *config) {
		c.streamKeyFormat = format
	}
}

func WithConsumerGroup(group string) Option {
	return func(c *config) {
		c.consumerGroup = group
	}
}

func WithProcessingTimeout(timeout time.Duration) Option {
	return func(c *config) {
		c.processingTimeout = timeout
	}
}

func WithClaimInterval(interval time.Duration) Option {
	return func(c *config) {
		c.claimInterval = interval
	}
}

func WithMaxClaimBatch(n int) Option {
	return func(c *config) {
		if n > 0 {
			c.maxClaimBatch = n
		}
	}
}

func WithBlockTimeout(timeout time.Duration) Option {
	return func(c *config) {
		c.blockTimeout = timeout
	}
}

func WithMaxStreamLength(maxLen int64) Option {
	return func(c *config) {
		c.maxStreamLength = maxLen
	}
}

func WithApproximateTrimming(enabled bool) Option {
	return func(c *config) {
		c.approximateTrim = enabled
	}
}

func WithClaim(enabled bool) Option {
	return func(c *config) {
		c.enableClaim = enabled
	}
}

func WithConsumerPrefix(prefix string) Option {
	return func(c *config) {
		c.consumerPrefix = prefix
	}
}
