package commandbus

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/shuldan/commands"
	"github.com/shuldan/queue"
)

// ResultDeserializer преобразует payload в результат.
type ResultDeserializer func(
	payload []byte, env *ResultEnvelope,
) (commands.Result, error)

// ResultCallbackFunc обрабатывает результат выполнения команды.
type ResultCallbackFunc func(
	ctx context.Context, result commands.Result, err error,
) error

type replyEntry struct {
	deserializer ResultDeserializer
	handler      ResultCallbackFunc
}

// ReplyListener слушает топик ответов и маршрутизирует результаты.
type ReplyListener struct {
	broker      queue.Broker
	logger      Logger
	serviceName string

	mu      sync.RWMutex
	entries map[string]*replyEntry
}

// NewReplyListener создаёт ReplyListener.
func NewReplyListener(
	broker queue.Broker,
	log Logger,
	opts ...ReplyListenerOption,
) *ReplyListener {
	cfg := &replyListenerConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	return &ReplyListener{
		broker:      broker,
		logger:      ensureLogger(log),
		serviceName: cfg.serviceName,
		entries:     make(map[string]*replyEntry),
	}
}

// OnResult регистрирует обработчик результата для типа команды.
func (l *ReplyListener) OnResult(
	commandName string,
	deserializer ResultDeserializer,
	handler ResultCallbackFunc,
) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.entries[commandName] = &replyEntry{
		deserializer: deserializer,
		handler:      handler,
	}

	l.logger.Info("reply listener: result handler registered",
		"command", commandName,
	)
}

// Run запускает consumer для топика ответов.
func (l *ReplyListener) Run(ctx context.Context) error {
	topic := replyTopicPrefix + l.serviceName

	l.logger.Info("reply listener: consuming",
		"topic", topic,
	)

	return l.broker.Consume(ctx, topic, func(data []byte) error {
		return l.handleMessage(ctx, data)
	})
}

func (l *ReplyListener) handleMessage(
	ctx context.Context,
	data []byte,
) error {
	env, err := unmarshalResultEnvelope(data)
	if err != nil {
		l.logger.Error("reply listener: unmarshal failed",
			"error", err,
		)

		return nil
	}

	l.mu.RLock()
	entry, ok := l.entries[env.CommandName]
	l.mu.RUnlock()

	if !ok {
		l.logger.Error("reply listener: no result handler",
			"command", env.CommandName,
			"correlation_id", env.CorrelationID,
		)

		return nil
	}

	return l.processResult(ctx, env, entry)
}

func (l *ReplyListener) processResult(
	ctx context.Context,
	env *ResultEnvelope,
	entry *replyEntry,
) error {
	if env.Error != nil {
		callbackErr := entry.handler(
			ctx, nil, errors.New(*env.Error),
		)

		if callbackErr != nil {
			l.logger.Error("reply listener: result handler error",
				"command", env.CommandName,
				"correlation_id", env.CorrelationID,
				"error", callbackErr,
			)
		}

		return nil
	}

	result, err := entry.deserializer(env.Payload, env)
	if err != nil {
		l.logger.Error("reply listener: deserialize result failed",
			"command", env.CommandName,
			"correlation_id", env.CorrelationID,
			"error", err,
		)

		return nil
	}

	if callbackErr := entry.handler(ctx, result, nil); callbackErr != nil {
		return fmt.Errorf(
			"reply listener: handler %s: %w",
			env.CommandName, callbackErr,
		)
	}

	return nil
}
