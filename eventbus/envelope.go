package eventbus

import (
	"encoding/json"
	"time"

	"github.com/shuldan/events"
)

// Envelope — стандартный конверт межсервисного события.
type Envelope struct {
	EventName   string          `json:"event_name"`
	AggregateID string          `json:"aggregate_id"`
	OccurredAt  time.Time       `json:"occurred_at"`
	Source      string          `json:"source,omitempty"`
	Payload     json.RawMessage `json:"payload"`

	// Зарезервированы для будущих расширений.
	CorrelationID string `json:"correlation_id,omitempty"`
	CausationID   string `json:"causation_id,omitempty"`
	SchemaVersion string `json:"schema_version,omitempty"`
}

func newEnvelope(
	event events.Event, payload json.RawMessage, source string,
) *Envelope {
	return &Envelope{
		EventName:   event.EventName(),
		AggregateID: event.AggregateID(),
		OccurredAt:  event.OccurredAt(),
		Source:      source,
		Payload:     payload,
	}
}

func marshalEnvelope(env *Envelope) ([]byte, error) {
	return json.Marshal(env)
}

func unmarshalEnvelope(data []byte) (*Envelope, error) {
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}

	return &env, nil
}
