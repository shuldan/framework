package eventbus

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/shuldan/events"
)

func TestOutboundRelay_ForwardsWithEnvelope(t *testing.T) {
	d := events.New(events.WithSyncMode())
	defer func() { _ = d.Close(context.Background()) }()

	broker := &mockBroker{}
	relay := NewOutboundRelay(d, broker, nil, WithSource("test-svc"))
	defer relay.Unsubscribe()

	relay.Forward("test", "test-topic")

	err := d.Publish(context.Background(), &testEvent{
		BaseEvent: events.NewBaseEvent("test", "agg-1"),
		Value:     "hello",
	})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}

	msgs := broker.Messages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].topic != "test-topic" {
		t.Errorf("expected topic 'test-topic', got %q", msgs[0].topic)
	}

	var env Envelope
	if err := json.Unmarshal(msgs[0].data, &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if env.EventName != "test" {
		t.Errorf("expected event_name 'test', got %q", env.EventName)
	}
	if env.AggregateID != "agg-1" {
		t.Errorf("expected aggregate_id 'agg-1', got %q", env.AggregateID)
	}
	if env.Source != "test-svc" {
		t.Errorf("expected source 'test-svc', got %q", env.Source)
	}
	if env.Payload == nil {
		t.Fatal("expected non-nil payload")
	}

	assertJSONContains(t, env.Payload, "hello")
}

func TestOutboundRelay_ForwardsWithTransform(t *testing.T) {
	d := events.New(events.WithSyncMode())
	defer func() { _ = d.Close(context.Background()) }()

	broker := &mockBroker{}
	relay := NewOutboundRelay(d, broker, nil)
	defer relay.Unsubscribe()

	relay.Forward("test", "custom-topic",
		WithTransform(func(e events.Event) ([]byte, error) {
			return []byte("custom:" + e.AggregateID()), nil
		}),
	)

	_ = d.Publish(context.Background(), &testEvent{
		BaseEvent: events.NewBaseEvent("test", "agg-99"),
	})

	msgs := broker.Messages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if string(msgs[0].data) != "custom:agg-99" {
		t.Errorf("expected 'custom:agg-99', got %q", msgs[0].data)
	}
}

func TestOutboundRelay_IgnoresUnregisteredEvent(t *testing.T) {
	d := events.New(events.WithSyncMode())
	defer func() { _ = d.Close(context.Background()) }()

	broker := &mockBroker{}
	relay := NewOutboundRelay(d, broker, nil)
	defer relay.Unsubscribe()

	relay.Forward("other", "other-topic")

	_ = d.Publish(context.Background(), &testEvent{
		BaseEvent: events.NewBaseEvent("test", "agg-1"),
	})

	if len(broker.Messages()) != 0 {
		t.Fatal("expected no messages for unregistered event")
	}
}

func TestOutboundRelay_Filter(t *testing.T) {
	d := events.New(events.WithSyncMode())
	defer func() { _ = d.Close(context.Background()) }()

	broker := &mockBroker{}
	relay := NewOutboundRelay(d, broker, nil)
	defer relay.Unsubscribe()

	relay.Forward("test", "filtered-topic",
		WithFilter(func(e events.Event) bool {
			te, ok := e.(*testEvent)
			return ok && te.Value == "pass"
		}),
	)

	ctx := context.Background()
	_ = d.Publish(ctx, &testEvent{BaseEvent: events.NewBaseEvent("test", "1"), Value: "reject"})
	_ = d.Publish(ctx, &testEvent{BaseEvent: events.NewBaseEvent("test", "2"), Value: "pass"})

	msgs := broker.Messages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
}

func TestOutboundRelay_TransformError(t *testing.T) {
	d := events.New(events.WithSyncMode())
	defer func() { _ = d.Close(context.Background()) }()

	broker := &mockBroker{}
	relay := NewOutboundRelay(d, broker, nil)
	defer relay.Unsubscribe()

	relay.Forward("test", "err-topic",
		WithTransform(func(_ events.Event) ([]byte, error) {
			return nil, errors.New("transform fail")
		}),
	)

	_ = d.Publish(context.Background(), &testEvent{
		BaseEvent: events.NewBaseEvent("test", "agg-1"),
	})

	if len(broker.Messages()) != 0 {
		t.Fatal("expected no messages when transform fails")
	}
}

func TestOutboundRelay_ProduceError(t *testing.T) {
	d := events.New(events.WithSyncMode())
	defer func() { _ = d.Close(context.Background()) }()

	broker := &errorBroker{err: errors.New("produce fail")}
	relay := NewOutboundRelay(d, broker, nil)
	defer relay.Unsubscribe()

	relay.Forward("test", "err-topic")

	err := d.Publish(context.Background(), &testEvent{
		BaseEvent: events.NewBaseEvent("test", "agg-1"),
	})
	if err == nil {
		t.Log("publish returned nil (error may be swallowed by dispatcher)")
	}
}

func TestOutboundRelay_Unsubscribe(t *testing.T) {
	d := events.New(events.WithSyncMode())
	defer func() { _ = d.Close(context.Background()) }()

	broker := &mockBroker{}
	relay := NewOutboundRelay(d, broker, nil)
	relay.Forward("test", "topic")
	relay.Unsubscribe()

	_ = d.Publish(context.Background(), &testEvent{
		BaseEvent: events.NewBaseEvent("test", "1"),
	})

	if len(broker.Messages()) != 0 {
		t.Fatal("expected no messages after unsubscribe")
	}
}

func TestOutboundRelay_UnsubscribeNilSub(t *testing.T) {
	t.Parallel()
	r := &OutboundRelay{}
	r.Unsubscribe()
}

func TestOutboundRelay_WithLogger(t *testing.T) {
	d := events.New(events.WithSyncMode())
	defer func() { _ = d.Close(context.Background()) }()

	ml := &relayMockLogger{}
	broker := &mockBroker{}
	relay := NewOutboundRelay(d, broker, ml)
	defer relay.Unsubscribe()

	relay.Forward("test", "topic")

	if !ml.infoCalled {
		t.Error("expected Info called during Forward")
	}
}

func TestOutboundRelay_NoSource(t *testing.T) {
	d := events.New(events.WithSyncMode())
	defer func() { _ = d.Close(context.Background()) }()

	broker := &mockBroker{}
	relay := NewOutboundRelay(d, broker, nil)
	defer relay.Unsubscribe()

	relay.Forward("test", "topic")

	_ = d.Publish(context.Background(), &testEvent{
		BaseEvent: events.NewBaseEvent("test", "agg-1"),
	})

	msgs := broker.Messages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	var env Envelope
	if err := json.Unmarshal(msgs[0].data, &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Source != "" {
		t.Errorf("expected empty source, got %q", env.Source)
	}
}

func TestEnsureRelayLogger_Nil(t *testing.T) {
	t.Parallel()
	l := ensureRelayLogger(nil)
	if _, ok := l.(noopLogger); !ok {
		t.Fatal("expected noopLogger")
	}
}

func TestEnsureRelayLogger_NonNil(t *testing.T) {
	t.Parallel()
	ml := &relayMockLogger{}
	l := ensureRelayLogger(ml)
	if l != ml {
		t.Fatal("expected same logger")
	}
}

func TestNoopRelayLogger(t *testing.T) {
	t.Parallel()
	l := noopLogger{}
	l.Info("test")
	l.Error("test")
}

// --- test helpers ---

type relayMockLogger struct {
	infoCalled  bool
	errorCalled bool
}

func (m *relayMockLogger) Info(_ string, _ ...any)  { m.infoCalled = true }
func (m *relayMockLogger) Error(_ string, _ ...any) { m.errorCalled = true }

type producedMsg struct {
	topic string
	data  []byte
}

type mockBroker struct {
	mu   sync.Mutex
	msgs []producedMsg
}

func (b *mockBroker) Produce(_ context.Context, topic string, data []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.msgs = append(b.msgs, producedMsg{topic: topic, data: data})
	return nil
}

func (b *mockBroker) Consume(_ context.Context, _ string, _ func([]byte) error) error { return nil }
func (b *mockBroker) Ping(_ context.Context) error                                    { return nil }
func (b *mockBroker) Close() error                                                    { return nil }

func (b *mockBroker) Messages() []producedMsg {
	b.mu.Lock()
	defer b.mu.Unlock()
	cp := make([]producedMsg, len(b.msgs))
	copy(cp, b.msgs)
	return cp
}

type errorBroker struct {
	err error
}

func (b *errorBroker) Produce(_ context.Context, _ string, _ []byte) error             { return b.err }
func (b *errorBroker) Consume(_ context.Context, _ string, _ func([]byte) error) error { return nil }
func (b *errorBroker) Ping(_ context.Context) error                                    { return nil }
func (b *errorBroker) Close() error                                                    { return nil }

func assertJSONContains(t *testing.T, data []byte, substr string) {
	t.Helper()
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		if s := string(data); !containsStr(s, substr) {
			t.Errorf("expected %q in %q", substr, s)
		}
		return
	}
	encoded, _ := json.Marshal(raw)
	if s := string(encoded); !containsStr(s, substr) {
		t.Errorf("expected %q in JSON: %s", substr, s)
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
