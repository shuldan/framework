package commandbus

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/shuldan/commands"
	"github.com/shuldan/queue"

	"github.com/shuldan/framework/queueworker"
)

const (
	replyTopicPrefix = "replies."
	defaultIdemTTL   = 24 * time.Hour
)

// CommandDeserializer преобразует payload в команду.
type CommandDeserializer func(
	payload []byte, env *CommandEnvelope,
) (commands.Command, error)

// CommandHandler обрабатывает команду и возвращает результат.
type CommandHandler interface {
	Handle(ctx context.Context, cmd commands.Command) (commands.Result, error)
}

// CommandHandlerFunc — адаптер для использования функции как CommandHandler.
type CommandHandlerFunc func(
	ctx context.Context, cmd commands.Command,
) (commands.Result, error)

// Handle реализует интерфейс CommandHandler.
func (f CommandHandlerFunc) Handle(
	ctx context.Context, cmd commands.Command,
) (commands.Result, error) {
	return f(ctx, cmd)
}

type inboundEntry struct {
	deserializer CommandDeserializer
	handler      CommandHandler
	idemTTL      time.Duration
}

// CommandReceiver принимает команды из очереди и выполняет их.
type CommandReceiver struct {
	broker    queue.Broker
	logger    Logger
	idemStore commands.IdempotencyStore
	idemTTL   time.Duration

	mu      sync.RWMutex
	entries map[string]*inboundEntry
}

// NewCommandReceiver создаёт CommandReceiver.
func NewCommandReceiver(
	broker queue.Broker,
	log Logger,
	opts ...ReceiverOption,
) *CommandReceiver {
	cfg := &receiverConfig{
		idemTTL: defaultIdemTTL,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	store := cfg.idemStore
	if store == nil {
		store = commands.NewMemoryIdempotencyStore()
	}

	return &CommandReceiver{
		broker:    broker,
		logger:    ensureLogger(log),
		idemStore: store,
		idemTTL:   cfg.idemTTL,
		entries:   make(map[string]*inboundEntry),
	}
}

// Handle регистрирует обработчик для типа команды.
func (r *CommandReceiver) Handle(
	commandName string,
	deserializer CommandDeserializer,
	handler CommandHandler,
	opts ...HandleOption,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.entries[commandName]; exists {
		return fmt.Errorf(
			"%w: %s", commands.ErrHandlerExists, commandName,
		)
	}

	entry := &inboundEntry{
		deserializer: deserializer,
		handler:      handler,
		idemTTL:      r.idemTTL,
	}

	for _, opt := range opts {
		opt(entry)
	}

	r.entries[commandName] = entry

	r.logger.Info("command receiver: handler registered",
		"command", commandName,
		"topic", commandTopicPrefix+commandName,
	)

	return nil
}

// Registrations возвращает список регистраций для queueworker.
func (r *CommandReceiver) Registrations() []queueworker.Registration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	regs := make([]queueworker.Registration, 0, len(r.entries))

	for name := range r.entries {
		topic := commandTopicPrefix + name
		cmdName := name

		regs = append(regs, queueworker.Registration{
			Name: "command-" + cmdName,
			Run: func(ctx context.Context) error {
				return r.broker.Consume(
					ctx, topic, func(data []byte) error {
						return r.handleMessage(ctx, data)
					},
				)
			},
		})
	}

	return regs
}

func (r *CommandReceiver) handleMessage(
	ctx context.Context,
	data []byte,
) error {
	env, err := unmarshalCommandEnvelope(data)
	if err != nil {
		r.logger.Error("command receiver: unmarshal failed",
			"error", err,
		)

		return nil
	}

	if r.isExpired(env) {
		return r.handleExpired(ctx, env)
	}

	if r.isDuplicate(ctx, env) {
		return nil
	}

	r.mu.RLock()
	entry, ok := r.entries[env.CommandName]
	r.mu.RUnlock()

	if !ok {
		r.logger.Error("command receiver: no handler",
			"command", env.CommandName,
		)

		return nil
	}

	return r.executeAndReply(ctx, env, entry)
}

func (r *CommandReceiver) isExpired(env *CommandEnvelope) bool {
	if env.Timeout <= 0 {
		return false
	}

	return time.Now().After(env.CreatedAt.Add(env.Timeout))
}

func (r *CommandReceiver) handleExpired(
	ctx context.Context,
	env *CommandEnvelope,
) error {
	r.logger.Warn("command receiver: command expired",
		"command", env.CommandName,
		"idempotency_key", env.IdempotencyKey,
		"created_at", env.CreatedAt,
		"timeout", env.Timeout,
	)

	if env.ReplyTo == "" {
		return nil
	}

	expiredErr := fmt.Errorf(
		"command expired: exceeded timeout of %s", env.Timeout,
	)

	return r.sendResult(ctx, env, nil, expiredErr)
}

func (r *CommandReceiver) isDuplicate(
	ctx context.Context,
	env *CommandEnvelope,
) bool {
	exists, err := r.idemStore.Exists(ctx, env.IdempotencyKey)
	if err != nil {
		r.logger.Error("command receiver: idempotency check failed",
			"command", env.CommandName,
			"idempotency_key", env.IdempotencyKey,
			"error", err,
		)

		return false
	}

	if exists {
		r.logger.Debug("command receiver: duplicate command",
			"command", env.CommandName,
			"idempotency_key", env.IdempotencyKey,
		)

		return true
	}

	return false
}

func (r *CommandReceiver) executeAndReply(
	ctx context.Context,
	env *CommandEnvelope,
	entry *inboundEntry,
) error {
	cmd, err := entry.deserializer(env.Payload, env)
	if err != nil {
		r.logger.Error("command receiver: deserialize failed",
			"command", env.CommandName,
			"error", err,
		)

		return nil
	}

	result, handleErr := entry.handler.Handle(ctx, cmd)

	if handleErr != nil {
		r.logger.Error("command receiver: handler error",
			"command", env.CommandName,
			"idempotency_key", env.IdempotencyKey,
			"error", handleErr,
		)

		if env.ReplyTo != "" {
			_ = r.sendResult(ctx, env, nil, handleErr)
		}

		// Возвращаем ошибку — очередь не ACK-ает, redelivery.
		return fmt.Errorf(
			"command handler %s: %w", env.CommandName, handleErr,
		)
	}

	if markErr := r.idemStore.Mark(
		ctx, env.IdempotencyKey, entry.idemTTL,
	); markErr != nil {
		r.logger.Error("command receiver: idempotency mark failed",
			"command", env.CommandName,
			"idempotency_key", env.IdempotencyKey,
			"error", markErr,
		)
	}

	if env.ReplyTo != "" {
		if sendErr := r.sendResult(
			ctx, env, result, nil,
		); sendErr != nil {
			r.logger.Error("command receiver: send result failed",
				"command", env.CommandName,
				"reply_to", env.ReplyTo,
				"error", sendErr,
			)
		}
	}

	return nil
}

func (r *CommandReceiver) sendResult(
	ctx context.Context,
	cmdEnv *CommandEnvelope,
	result commands.Result,
	err error,
) error {
	resultEnv := &ResultEnvelope{
		CorrelationID: cmdEnv.CorrelationID,
		CommandName:   cmdEnv.CommandName,
		CreatedAt:     time.Now().UTC(),
		Error:         errorToPtr(err),
	}

	if result != nil {
		resultEnv.ResultName = result.ResultName()

		payload, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			return fmt.Errorf("marshal result payload: %w", marshalErr)
		}

		resultEnv.Payload = payload
	}

	data, marshalErr := marshalResultEnvelope(resultEnv)
	if marshalErr != nil {
		return fmt.Errorf("marshal result envelope: %w", marshalErr)
	}

	topic := replyTopicPrefix + cmdEnv.ReplyTo

	return r.broker.Produce(ctx, topic, data)
}
