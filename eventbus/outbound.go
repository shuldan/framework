package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/shuldan/events"
	"github.com/shuldan/queue"
)

type outboundEntry struct {
	topic     string
	filter    func(events.Event) bool
	transform func(events.Event) ([]byte, error)
}

// OutboundRelay пересылает внутренние события в очередь.
type OutboundRelay struct {
	broker  queue.Broker
	logger  Logger
	source  string
	mu      sync.RWMutex
	entries map[string]*outboundEntry
	sub     events.Subscription
}

// NewOutboundRelay создаёт OutboundRelay и подписывается на все события диспетчера.
func NewOutboundRelay(
	dispatcher *events.Dispatcher,
	broker queue.Broker,
	log Logger,
	opts ...OutboundConfig,
) *OutboundRelay {
	cfg := &outboundConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	r := &OutboundRelay{
		broker:  broker,
		logger:  ensureRelayLogger(log),
		source:  cfg.source,
		entries: make(map[string]*outboundEntry),
	}

	r.sub = dispatcher.SubscribeAll(r.handle)

	return r
}

// Forward регистрирует пересылку события eventName в топик topic.
func (r *OutboundRelay) Forward(
	eventName, topic string, opts ...OutboundOption,
) {
	entry := &outboundEntry{topic: topic}
	for _, opt := range opts {
		opt(entry)
	}

	r.mu.Lock()
	r.entries[eventName] = entry
	r.mu.Unlock()

	r.logger.Info("outbound relay registered",
		"event", eventName, "topic", topic,
	)
}

// Unsubscribe отписывает OutboundRelay от диспетчера.
func (r *OutboundRelay) Unsubscribe() {
	if r.sub != nil {
		r.sub.Unsubscribe()
	}
}

func (r *OutboundRelay) handle(
	ctx context.Context, event events.Event,
) error {
	r.mu.RLock()
	entry, ok := r.entries[event.EventName()]
	r.mu.RUnlock()

	if !ok {
		return nil
	}

	if entry.filter != nil && !entry.filter(event) {
		return nil
	}

	data, err := r.serialize(event, entry)
	if err != nil {
		r.logger.Error("outbound relay: serialize failed",
			"event", event.EventName(),
			"error", err,
		)

		return nil
	}

	if produceErr := r.broker.Produce(ctx, entry.topic, data); produceErr != nil {
		return fmt.Errorf(
			"outbound relay: produce to %q: %w",
			entry.topic, produceErr,
		)
	}

	return nil
}

func (r *OutboundRelay) serialize(
	event events.Event, entry *outboundEntry,
) ([]byte, error) {
	if entry.transform != nil {
		return entry.transform(event)
	}

	return r.serializeEnvelope(event)
}

func (r *OutboundRelay) serializeEnvelope(
	event events.Event,
) ([]byte, error) {
	payload, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	env := newEnvelope(event, payload, r.source)

	return marshalEnvelope(env)
}

func ensureRelayLogger(log Logger) Logger {
	if log == nil {
		return noopLogger{}
	}

	return log
}
