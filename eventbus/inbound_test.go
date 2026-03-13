package eventbus

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shuldan/events"
)

func TestInboundRelay_ReceivesAndPublishes(t *testing.T) {
	d := events.New(events.WithSyncMode())
	defer func() { _ = d.Close(context.Background()) }()

	var received atomic.Value
	events.SubscribeFunc(d, func(_ context.Context, e *testEvent) error {
		received.Store(e)
		return nil
	})

	broker := newInboundMockBroker()
	inbound := NewInboundRelay(d, broker, nil)

	inbound.On("task.completed", "TaskCompleted", func(payload []byte, _ *Envelope) (events.Event, error) {
		var data struct {
			TaskID string `json:"task_id"`
		}
		if err := json.Unmarshal(payload, &data); err != nil {
			return nil, err
		}
		return &testEvent{
			BaseEvent: events.NewBaseEvent("TaskCompleted", data.TaskID),
			Value:     data.TaskID,
		}, nil
	})

	env := &Envelope{
		EventName:   "TaskCompleted",
		AggregateID: "task-42",
		OccurredAt:  time.Now().UTC(),
		Payload:     json.RawMessage(`{"task_id":"task-42"}`),
	}
	envData, _ := marshalEnvelope(env)
	broker.Enqueue("task.completed", envData)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := inbound.RunTopic("task.completed")(ctx)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("RunTopic: %v", err)
	}

	val := received.Load()
	if val == nil {
		t.Fatal("expected event to be received")
	}

	te := val.(*testEvent)
	if te.Value != "task-42" {
		t.Errorf("expected Value 'task-42', got %q", te.Value)
	}
}

func TestInboundRelay_SkipsOwnEvents(t *testing.T) {
	d := events.New(events.WithSyncMode())
	defer func() { _ = d.Close(context.Background()) }()

	var called atomic.Bool
	events.SubscribeFunc(d, func(_ context.Context, _ *testEvent) error {
		called.Store(true)
		return nil
	})

	broker := newInboundMockBroker()
	inbound := NewInboundRelay(d, broker, nil, WithServiceName("my-service"))

	inbound.On("topic", "Evt", func(payload []byte, _ *Envelope) (events.Event, error) {
		return &testEvent{BaseEvent: events.NewBaseEvent("Evt", "a")}, nil
	})

	env := &Envelope{
		EventName:   "Evt",
		AggregateID: "a",
		OccurredAt:  time.Now().UTC(),
		Source:      "my-service",
		Payload:     json.RawMessage(`{}`),
	}
	envData, _ := marshalEnvelope(env)
	broker.Enqueue("topic", envData)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = inbound.RunTopic("topic")(ctx)

	if called.Load() {
		t.Fatal("expected own event to be skipped")
	}
}

func TestInboundRelay_DoesNotSkipForeignEvents(t *testing.T) {
	d := events.New(events.WithSyncMode())
	defer func() { _ = d.Close(context.Background()) }()

	var called atomic.Bool
	events.SubscribeFunc(d, func(_ context.Context, _ *testEvent) error {
		called.Store(true)
		return nil
	})

	broker := newInboundMockBroker()
	inbound := NewInboundRelay(d, broker, nil, WithServiceName("my-service"))

	inbound.On("topic", "Evt", func(_ []byte, _ *Envelope) (events.Event, error) {
		return &testEvent{BaseEvent: events.NewBaseEvent("Evt", "a")}, nil
	})

	env := &Envelope{
		EventName:   "Evt",
		AggregateID: "a",
		OccurredAt:  time.Now().UTC(),
		Source:      "other-service",
		Payload:     json.RawMessage(`{}`),
	}
	envData, _ := marshalEnvelope(env)
	broker.Enqueue("topic", envData)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_ = inbound.RunTopic("topic")(ctx)

	if !called.Load() {
		t.Fatal("expected foreign event to be processed")
	}
}

func TestInboundRelay_NoServiceName_ProcessesAll(t *testing.T) {
	d := events.New(events.WithSyncMode())
	defer func() { _ = d.Close(context.Background()) }()

	var called atomic.Bool
	events.SubscribeFunc(d, func(_ context.Context, _ *testEvent) error {
		called.Store(true)
		return nil
	})

	broker := newInboundMockBroker()
	inbound := NewInboundRelay(d, broker, nil)

	inbound.On("topic", "Evt", func(_ []byte, _ *Envelope) (events.Event, error) {
		return &testEvent{BaseEvent: events.NewBaseEvent("Evt", "a")}, nil
	})

	env := &Envelope{
		EventName:   "Evt",
		AggregateID: "a",
		OccurredAt:  time.Now().UTC(),
		Source:      "any-service",
		Payload:     json.RawMessage(`{}`),
	}
	envData, _ := marshalEnvelope(env)
	broker.Enqueue("topic", envData)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_ = inbound.RunTopic("topic")(ctx)

	if !called.Load() {
		t.Fatal("expected event to be processed without service filter")
	}
}

func TestInboundRelay_InvalidEnvelope(t *testing.T) {
	d := events.New(events.WithSyncMode())
	defer func() { _ = d.Close(context.Background()) }()

	broker := newInboundMockBroker()
	ml := &relayMockLogger{}
	inbound := NewInboundRelay(d, broker, ml)

	inbound.On("topic", "Evt", func(_ []byte, _ *Envelope) (events.Event, error) {
		return &testEvent{BaseEvent: events.NewBaseEvent("Evt", "a")}, nil
	})

	broker.Enqueue("topic", []byte(`{invalid json`))

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = inbound.RunTopic("topic")(ctx)

	if !ml.errorCalled {
		t.Error("expected error to be logged for invalid envelope")
	}
}

func TestInboundRelay_NoDeserializer(t *testing.T) {
	d := events.New(events.WithSyncMode())
	defer func() { _ = d.Close(context.Background()) }()

	broker := newInboundMockBroker()
	ml := &relayMockLogger{}
	inbound := NewInboundRelay(d, broker, ml)

	env := &Envelope{
		EventName:   "UnknownEvent",
		AggregateID: "a",
		OccurredAt:  time.Now().UTC(),
		Payload:     json.RawMessage(`{}`),
	}
	envData, _ := marshalEnvelope(env)
	broker.Enqueue("topic", envData)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = inbound.RunTopic("topic")(ctx)

	if !ml.errorCalled {
		t.Error("expected error for missing deserializer")
	}
}

func TestInboundRelay_DeserializerError(t *testing.T) {
	d := events.New(events.WithSyncMode())
	defer func() { _ = d.Close(context.Background()) }()

	broker := newInboundMockBroker()
	ml := &relayMockLogger{}
	inbound := NewInboundRelay(d, broker, ml)

	inbound.On("topic", "Evt", func(_ []byte, _ *Envelope) (events.Event, error) {
		return nil, errors.New("bad payload")
	})

	env := &Envelope{
		EventName:   "Evt",
		AggregateID: "a",
		OccurredAt:  time.Now().UTC(),
		Payload:     json.RawMessage(`{}`),
	}
	envData, _ := marshalEnvelope(env)
	broker.Enqueue("topic", envData)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_ = inbound.RunTopic("topic")(ctx)

	if !ml.errorCalled {
		t.Error("expected error for deserializer failure")
	}
}

func TestInboundRelay_Topics(t *testing.T) {
	t.Parallel()
	inbound := NewInboundRelay(nil, nil, nil)

	inbound.On("topic-a", "Evt1", nil)
	inbound.On("topic-b", "Evt2", nil)
	inbound.On("topic-a", "Evt3", nil)

	topics := inbound.Topics()
	if len(topics) != 2 {
		t.Fatalf("expected 2 topics, got %d", len(topics))
	}

	found := make(map[string]bool)
	for _, tp := range topics {
		found[tp] = true
	}
	if !found["topic-a"] || !found["topic-b"] {
		t.Errorf("expected topic-a and topic-b, got %v", topics)
	}
}

func TestInboundRelay_MultipleEventsOnSameTopic(t *testing.T) {
	d := events.New(events.WithSyncMode())
	defer func() { _ = d.Close(context.Background()) }()

	var count atomic.Int32
	events.SubscribeFunc(d, func(_ context.Context, _ *testEvent) error {
		count.Add(1)
		return nil
	})

	broker := newInboundMockBroker()
	inbound := NewInboundRelay(d, broker, nil)

	inbound.On("topic", "EvtA", func(_ []byte, _ *Envelope) (events.Event, error) {
		return &testEvent{BaseEvent: events.NewBaseEvent("EvtA", "a")}, nil
	})
	inbound.On("topic", "EvtB", func(_ []byte, _ *Envelope) (events.Event, error) {
		return &testEvent{BaseEvent: events.NewBaseEvent("EvtB", "b")}, nil
	})

	for _, name := range []string{"EvtA", "EvtB"} {
		env := &Envelope{
			EventName:   name,
			AggregateID: "x",
			OccurredAt:  time.Now().UTC(),
			Payload:     json.RawMessage(`{}`),
		}
		data, _ := marshalEnvelope(env)
		broker.Enqueue("topic", data)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_ = inbound.RunTopic("topic")(ctx)

	if count.Load() != 2 {
		t.Fatalf("expected 2 events handled, got %d", count.Load())
	}
}

func TestInboundRelay_WithLogger(t *testing.T) {
	t.Parallel()
	ml := &relayMockLogger{}
	inbound := NewInboundRelay(nil, nil, ml)
	inbound.On("t", "e", nil)

	if !ml.infoCalled {
		t.Error("expected Info called during On")
	}
}

// --- inbound test broker ---

type inboundMockBroker struct {
	mu       sync.Mutex
	messages map[string][][]byte
}

func newInboundMockBroker() *inboundMockBroker {
	return &inboundMockBroker{
		messages: make(map[string][][]byte),
	}
}

func (b *inboundMockBroker) Enqueue(topic string, data []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.messages[topic] = append(b.messages[topic], data)
}

func (b *inboundMockBroker) Produce(_ context.Context, topic string, data []byte) error {
	b.Enqueue(topic, data)
	return nil
}

func (b *inboundMockBroker) Consume(ctx context.Context, topic string, handler func([]byte) error) error {
	b.mu.Lock()
	msgs := b.messages[topic]
	b.messages[topic] = nil
	b.mu.Unlock()

	for _, msg := range msgs {
		if err := handler(msg); err != nil {
			return err
		}
	}

	<-ctx.Done()
	return ctx.Err()
}

func (b *inboundMockBroker) Ping(_ context.Context) error { return nil }
func (b *inboundMockBroker) Close() error                 { return nil }
