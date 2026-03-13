package eventbus

import (
	"context"
	"fmt"
	"sync"

	"github.com/shuldan/events"
	"github.com/shuldan/queue"
)

// Deserializer преобразует payload из envelope в доменное событие.
type Deserializer func(payload []byte, envelope *Envelope) (events.Event, error)

type inboundEntry struct {
	deserializer Deserializer
}

// InboundRelay слушает топики брокера и публикует события в Dispatcher.
type InboundRelay struct {
	dispatcher *events.Dispatcher
	broker     queue.Broker
	logger     Logger
	service    string

	mu     sync.RWMutex
	topics map[string]map[string]*inboundEntry // topic → eventName → entry
}

// NewInboundRelay создаёт InboundRelay.
func NewInboundRelay(
	dispatcher *events.Dispatcher,
	broker queue.Broker,
	log Logger,
	opts ...InboundOption,
) *InboundRelay {
	cfg := &inboundConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	return &InboundRelay{
		dispatcher: dispatcher,
		broker:     broker,
		logger:     ensureRelayLogger(log),
		service:    cfg.service,
		topics:     make(map[string]map[string]*inboundEntry),
	}
}

// On регистрирует десериализатор для события eventName из топика topic.
func (r *InboundRelay) On(
	topic, eventName string, deserializer Deserializer,
) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.topics[topic] == nil {
		r.topics[topic] = make(map[string]*inboundEntry)
	}

	r.topics[topic][eventName] = &inboundEntry{
		deserializer: deserializer,
	}

	r.logger.Info("inbound relay registered",
		"topic", topic, "event", eventName,
	)
}

// Topics возвращает список зарегистрированных топиков.
func (r *InboundRelay) Topics() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	topics := make([]string, 0, len(r.topics))
	for t := range r.topics {
		topics = append(topics, t)
	}

	return topics
}

// RunTopic возвращает функцию запуска consumer-а для конкретного топика.
// Совместима с queueworker.Registration.Run.
func (r *InboundRelay) RunTopic(topic string) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		r.logger.Info("inbound relay consuming",
			"topic", topic,
		)

		return r.broker.Consume(ctx, topic, func(data []byte) error {
			return r.handleMessage(ctx, topic, data)
		})
	}
}

func (r *InboundRelay) handleMessage(
	ctx context.Context, topic string, data []byte,
) error {
	env, err := unmarshalEnvelope(data)
	if err != nil {
		r.logger.Error("inbound relay: unmarshal envelope failed",
			"topic", topic,
			"error", err,
		)

		return nil
	}

	if r.shouldSkip(env) {
		return nil
	}

	r.mu.RLock()
	topicEntries := r.topics[topic]
	var entry *inboundEntry
	if topicEntries != nil {
		entry = topicEntries[env.EventName]
	}
	r.mu.RUnlock()

	if entry == nil {
		r.logger.Error("inbound relay: no deserializer",
			"topic", topic,
			"event_name", env.EventName,
		)

		return nil
	}

	event, err := entry.deserializer(env.Payload, env)
	if err != nil {
		r.logger.Error("inbound relay: deserialize failed",
			"topic", topic,
			"event_name", env.EventName,
			"error", err,
		)

		return nil
	}

	if publishErr := r.dispatcher.Publish(ctx, event); publishErr != nil {
		return fmt.Errorf(
			"inbound relay: publish %q: %w",
			env.EventName, publishErr,
		)
	}

	return nil
}

func (r *InboundRelay) shouldSkip(env *Envelope) bool {
	if r.service != "" && env.Source == r.service {
		r.logger.Info("inbound relay: skipping own event",
			"event_name", env.EventName,
			"source", env.Source,
		)

		return true
	}

	return false
}
