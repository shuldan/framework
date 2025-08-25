package queue

import (
	"github.com/shuldan/framework/pkg/contracts"
	"reflect"
)

func NewDefaultErrorHandler(logger contracts.Logger) ErrorHandler {
	return &defaultErrorHandler{
		logger: logger,
	}
}

func NewDefaultPanicHandler(logger contracts.Logger) PanicHandler {
	return &defaultPanicHandler{
		logger: logger,
	}
}

func New[T any](broker Broker, opts ...Option) (Queue[T], error) {
	var t T
	typ := reflect.TypeOf(t)

	if typ.Kind() != reflect.Ptr || typ.Elem().Kind() != reflect.Struct {
		return nil, ErrInvalidJobType.WithDetail("type", typ.String())
	}

	config := &queueConfig{
		broker:       broker,
		concurrency:  1,
		maxRetries:   0,
		backoff:      NoBackoff{},
		errorHandler: NewDefaultErrorHandler(nil),
		panicHandler: NewDefaultPanicHandler(nil),
		prefix:       "",
		dlqEnabled:   false,
		counter:      NoOpCounter{},
	}

	for _, opt := range opts {
		opt(config)
	}

	topic := typ.String()

	q := &typedQueue[T]{
		topic:        topic,
		broker:       config.broker,
		backoff:      config.backoff,
		concurrency:  config.concurrency,
		maxRetries:   config.maxRetries,
		panicHandler: config.panicHandler,
		errorHandler: config.errorHandler,
		prefix:       config.prefix,
		dlqEnabled:   config.dlqEnabled,
		counter:      config.counter,
	}

	return q, nil
}
