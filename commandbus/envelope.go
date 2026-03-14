package commandbus

import (
	"encoding/json"
	"time"
)

// CommandEnvelope — конверт команды для межсервисной доставки.
type CommandEnvelope struct {
	IdempotencyKey string            `json:"idempotency_key"`
	CommandName    string            `json:"command_name"`
	ReplyTo        string            `json:"reply_to,omitempty"`
	CorrelationID  string            `json:"correlation_id"`
	Sender         string            `json:"sender,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	Timeout        time.Duration     `json:"timeout"`
	Payload        json.RawMessage   `json:"payload"`
	SchemaVersion  string            `json:"schema_version,omitempty"`
	Headers        map[string]string `json:"headers,omitempty"`
}

// ResultEnvelope — конверт результата выполнения команды.
type ResultEnvelope struct {
	CorrelationID string            `json:"correlation_id"`
	CommandName   string            `json:"command_name"`
	ResultName    string            `json:"result_name,omitempty"`
	Sender        string            `json:"sender,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	Payload       json.RawMessage   `json:"payload,omitempty"`
	Error         *string           `json:"error"`
	SchemaVersion string            `json:"schema_version,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
}

func marshalCommandEnvelope(env *CommandEnvelope) ([]byte, error) {
	return json.Marshal(env)
}

func unmarshalCommandEnvelope(data []byte) (*CommandEnvelope, error) {
	var env CommandEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}

	return &env, nil
}

func marshalResultEnvelope(env *ResultEnvelope) ([]byte, error) {
	return json.Marshal(env)
}

func unmarshalResultEnvelope(data []byte) (*ResultEnvelope, error) {
	var env ResultEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}

	return &env, nil
}

func errorToPtr(err error) *string {
	if err == nil {
		return nil
	}

	s := err.Error()

	return &s
}
