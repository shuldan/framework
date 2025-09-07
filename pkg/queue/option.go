package queue

type Option func(*queueConfig)

type queueConfig struct {
	broker       Broker
	errorHandler ErrorHandler
	panicHandler PanicHandler
	concurrency  int
	maxRetries   int
	backoff      BackoffStrategy
	prefix       string
	dlqEnabled   bool
	counter      Counter
}

func WithPrefix(prefix string) Option {
	return func(q *queueConfig) {
		q.prefix = prefix
	}
}

func WithDLQ(enabled bool) Option {
	return func(q *queueConfig) {
		q.dlqEnabled = enabled
	}
}

func WithConcurrency(n int) Option {
	return func(q *queueConfig) {
		if n < 1 {
			n = 1
		}
		q.concurrency = n
	}
}

func WithMaxRetries(n int) Option {
	return func(q *queueConfig) {
		q.maxRetries = n
	}
}

func WithBackoff(b BackoffStrategy) Option {
	return func(q *queueConfig) {
		q.backoff = b
	}
}

func WithErrorHandler(h ErrorHandler) Option {
	return func(q *queueConfig) {
		q.errorHandler = h
	}
}

func WithPanicHandler(h PanicHandler) Option {
	return func(q *queueConfig) {
		q.panicHandler = h
	}
}

func WithCounter(counter Counter) Option {
	return func(q *queueConfig) {
		q.counter = counter
	}
}
