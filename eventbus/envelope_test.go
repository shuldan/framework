package eventbus

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/shuldan/events"
)

func TestNewEnvelope(t *testing.T) {
	t.Parallel()
	base := events.NewBaseEvent("TestEvent", "agg-1")
	payload := json.RawMessage(`{"key":"value"}`)

	env := newEnvelope(&testEvent{BaseEvent: base}, payload, "order-service")

	if env.EventName != "TestEvent" {
		t.Errorf("expected EventName 'TestEvent', got %q", env.EventName)
	}
	if env.AggregateID != "agg-1" {
		t.Errorf("expected AggregateID 'agg-1', got %q", env.AggregateID)
	}
	if env.Source != "order-service" {
		t.Errorf("expected Source 'order-service', got %q", env.Source)
	}
	if string(env.Payload) != `{"key":"value"}` {
		t.Errorf("expected payload, got %q", env.Payload)
	}
	if env.OccurredAt.IsZero() {
		t.Error("expected non-zero OccurredAt")
	}
}

func TestNewEnvelope_EmptySource(t *testing.T) {
	t.Parallel()
	base := events.NewBaseEvent("Evt", "agg-2")
	env := newEnvelope(&testEvent{BaseEvent: base}, nil, "")

	if env.Source != "" {
		t.Errorf("expected empty source, got %q", env.Source)
	}
}

func TestMarshalUnmarshalEnvelope(t *testing.T) {
	t.Parallel()
	original := &Envelope{
		EventName:   "TaskCreated",
		AggregateID: "agg-42",
		OccurredAt:  time.Now().UTC().Truncate(time.Millisecond),
		Source:      "task-service",
		Payload:     json.RawMessage(`{"task_id":"42"}`),
	}

	data, err := marshalEnvelope(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	restored, err := unmarshalEnvelope(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if restored.EventName != original.EventName {
		t.Errorf("EventName: expected %q, got %q", original.EventName, restored.EventName)
	}
	if restored.AggregateID != original.AggregateID {
		t.Errorf("AggregateID: expected %q, got %q", original.AggregateID, restored.AggregateID)
	}
	if restored.Source != original.Source {
		t.Errorf("Source: expected %q, got %q", original.Source, restored.Source)
	}
	if string(restored.Payload) != string(original.Payload) {
		t.Errorf("Payload: expected %q, got %q", original.Payload, restored.Payload)
	}
	if !restored.OccurredAt.Equal(original.OccurredAt) {
		t.Errorf("OccurredAt: expected %v, got %v", original.OccurredAt, restored.OccurredAt)
	}
}

func TestUnmarshalEnvelope_InvalidJSON(t *testing.T) {
	t.Parallel()
	_, err := unmarshalEnvelope([]byte(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestMarshalEnvelope_ReservedFieldsOmitted(t *testing.T) {
	t.Parallel()
	env := &Envelope{
		EventName:   "Evt",
		AggregateID: "a",
		OccurredAt:  time.Now().UTC(),
		Payload:     json.RawMessage(`{}`),
	}

	data, err := marshalEnvelope(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]any
	_ = json.Unmarshal(data, &raw)

	for _, key := range []string{"correlation_id", "causation_id", "schema_version", "source"} {
		if _, ok := raw[key]; ok {
			t.Errorf("expected %q to be omitted from JSON", key)
		}
	}
}

func TestMarshalEnvelope_ReservedFieldsPresent(t *testing.T) {
	t.Parallel()
	env := &Envelope{
		EventName:     "Evt",
		AggregateID:   "a",
		OccurredAt:    time.Now().UTC(),
		Payload:       json.RawMessage(`{}`),
		CorrelationID: "corr-1",
		CausationID:   "cause-1",
		SchemaVersion: "v2",
	}

	data, err := marshalEnvelope(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]any
	_ = json.Unmarshal(data, &raw)

	if raw["correlation_id"] != "corr-1" {
		t.Errorf("expected correlation_id 'corr-1', got %v", raw["correlation_id"])
	}
	if raw["causation_id"] != "cause-1" {
		t.Errorf("expected causation_id 'cause-1', got %v", raw["causation_id"])
	}
	if raw["schema_version"] != "v2" {
		t.Errorf("expected schema_version 'v2', got %v", raw["schema_version"])
	}
}

func TestUnmarshalEnvelope_IgnoresUnknownFields(t *testing.T) {
	t.Parallel()
	data := []byte(`{
		"event_name": "Evt",
		"aggregate_id": "a",
		"occurred_at": "2025-01-01T00:00:00Z",
		"payload": {},
		"future_field": "hello"
	}`)

	env, err := unmarshalEnvelope(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.EventName != "Evt" {
		t.Errorf("expected 'Evt', got %q", env.EventName)
	}
}

func TestEnvelope_NilPayload(t *testing.T) {
	t.Parallel()
	env := &Envelope{
		EventName:   "Evt",
		AggregateID: "a",
		OccurredAt:  time.Now().UTC(),
		Payload:     nil,
	}

	data, err := marshalEnvelope(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	restored, err := unmarshalEnvelope(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if restored.Payload != nil && string(restored.Payload) != "null" {
		t.Errorf("expected nil or null payload, got %q", restored.Payload)
	}
}
