package commandbus

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/shuldan/commands"
	"github.com/shuldan/queue"
)

const (
	commandTopicPrefix = "commands."
	defaultTimeout     = 30 * time.Second
)

// CommandSender отправляет команды в очередь для межсервисного взаимодействия.
type CommandSender struct {
	broker  queue.Broker
	logger  Logger
	replyTo string
	sender  string
	timeout time.Duration
	routes  map[string]struct{}
}

// NewCommandSender создаёт CommandSender.
func NewCommandSender(
	broker queue.Broker,
	log Logger,
	opts ...SenderOption,
) *CommandSender {
	cfg := &senderConfig{
		timeout: defaultTimeout,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return &CommandSender{
		broker:  broker,
		logger:  ensureLogger(log),
		replyTo: cfg.replyTo,
		sender:  cfg.sender,
		timeout: cfg.timeout,
		routes:  make(map[string]struct{}),
	}
}

// Forward регистрирует маршрут для команды.
func (s *CommandSender) Forward(commandName string) {
	s.routes[commandName] = struct{}{}

	s.logger.Info("command sender: route registered",
		"command", commandName,
		"topic", commandTopicPrefix+commandName,
	)
}

// Send отправляет команду в очередь.
func (s *CommandSender) Send(
	ctx context.Context,
	cmd commands.Command,
	opts ...SendOption,
) error {
	if cmd == nil {
		return commands.ErrNilCommand
	}

	name := cmd.CommandName()

	if _, ok := s.routes[name]; !ok {
		return fmt.Errorf(
			"%w: %s", commands.ErrHandlerNotFound, name,
		)
	}

	so := &sendOptions{
		timeout: s.timeout,
		replyTo: s.replyTo,
	}

	for _, opt := range opts {
		opt(so)
	}

	env, err := s.buildEnvelope(cmd, so)
	if err != nil {
		return fmt.Errorf("command sender: build envelope: %w", err)
	}

	data, err := marshalCommandEnvelope(env)
	if err != nil {
		return fmt.Errorf("command sender: marshal: %w", err)
	}

	topic := commandTopicPrefix + name

	if err = s.broker.Produce(ctx, topic, data); err != nil {
		return fmt.Errorf(
			"command sender: produce to %q: %w", topic, err,
		)
	}

	s.logger.Info("command sender: sent",
		"command", name,
		"topic", topic,
		"idempotency_key", env.IdempotencyKey,
		"correlation_id", env.CorrelationID,
	)

	return nil
}

func (s *CommandSender) buildEnvelope(
	cmd commands.Command,
	so *sendOptions,
) (*CommandEnvelope, error) {
	payload, err := json.Marshal(cmd)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	key := cmd.IdempotencyKey()
	if key == "" {
		key = uuid.New().String()
	}

	return &CommandEnvelope{
		IdempotencyKey: key,
		CommandName:    cmd.CommandName(),
		ReplyTo:        so.replyTo,
		CorrelationID:  uuid.New().String(),
		Sender:         s.sender,
		CreatedAt:      time.Now().UTC(),
		Timeout:        so.timeout,
		Payload:        payload,
		Headers:        so.headers,
	}, nil
}
