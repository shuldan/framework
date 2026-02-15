package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/shuldan/events"
	"github.com/shuldan/queue"
)

type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

type noopLogger struct{}

func (noopLogger) Info(string, ...any)  {}
func (noopLogger) Error(string, ...any) {}

type RelayOption func(*relayEntry)

func WithFilter(fn func(events.Event) bool) RelayOption {
	return func(e *relayEntry) {
		e.filter = fn
	}
}

func WithTransform(
	fn func(events.Event) ([]byte, error),
) RelayOption {
	return func(e *relayEntry) {
		e.transform = fn
	}
}

type relayEntry struct {
	topic     string
	filter    func(events.Event) bool
	transform func(events.Event) ([]byte, error)
}

type Relay struct {
	broker  queue.Broker
	logger  Logger
	mu      sync.RWMutex
	entries map[string]*relayEntry
	sub     events.Subscription
}

func NewRelay(
	dispatcher *events.Dispatcher,
	broker queue.Broker,
	log Logger,
) *Relay {
	r := &Relay{
		broker:  broker,
		logger:  ensureRelayLogger(log),
		entries: make(map[string]*relayEntry),
	}

	r.sub = dispatcher.SubscribeAll(r.handle)

	return r
}

func (r *Relay) Forward(
	eventName, topic string, opts ...RelayOption,
) {
	entry := &relayEntry{topic: topic}
	for _, opt := range opts {
		opt(entry)
	}

	r.mu.Lock()
	r.entries[eventName] = entry
	r.mu.Unlock()

	r.logger.Info("relay registered",
		"event", eventName, "topic", topic,
	)
}

func (r *Relay) Unsubscribe() {
	if r.sub != nil {
		r.sub.Unsubscribe()
	}
}

func (r *Relay) handle(
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
		r.logger.Error("relay: serialize failed",
			"event", event.EventName(),
			"error", err,
		)

		return nil
	}

	if produceErr := r.broker.Produce(ctx, entry.topic, data); produceErr != nil {
		return fmt.Errorf(
			"relay: produce to %q: %w", entry.topic, produceErr,
		)
	}

	return nil
}

func (r *Relay) serialize(
	event events.Event, entry *relayEntry,
) ([]byte, error) {
	if entry.transform != nil {
		return entry.transform(event)
	}

	return json.Marshal(event)
}

func ensureRelayLogger(log Logger) Logger {
	if log == nil {
		return noopLogger{}
	}

	return log
}
