package commandbus

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestMarshalCommandEnvelope_Success(t *testing.T) {
	t.Parallel()
	env := &CommandEnvelope{
		IdempotencyKey: "key-1",
		CommandName:    "test.cmd",
		ReplyTo:        "svc-a",
		CorrelationID:  "corr-1",
		CreatedAt:      time.Now().UTC(),
		Timeout:        5 * time.Second,
		Payload:        json.RawMessage(`{"foo":"bar"}`),
		SchemaVersion:  "v1",
		Headers:        map[string]string{"h1": "v1"},
	}
	data, err := marshalCommandEnvelope(env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty data")
	}
}

func TestUnmarshalCommandEnvelope_Success(t *testing.T) {
	t.Parallel()
	original := &CommandEnvelope{
		IdempotencyKey: "key-2",
		CommandName:    "test.cmd2",
		CorrelationID:  "corr-2",
		CreatedAt:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Timeout:        10 * time.Second,
		Payload:        json.RawMessage(`{"x":1}`),
	}
	data, _ := marshalCommandEnvelope(original)
	env, err := unmarshalCommandEnvelope(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.IdempotencyKey != original.IdempotencyKey {
		t.Errorf("expected key %q, got %q", original.IdempotencyKey, env.IdempotencyKey)
	}
	if env.CommandName != original.CommandName {
		t.Errorf("expected name %q, got %q", original.CommandName, env.CommandName)
	}
}

func TestUnmarshalCommandEnvelope_InvalidJSON(t *testing.T) {
	t.Parallel()
	_, err := unmarshalCommandEnvelope([]byte("not-json"))
	if err == nil {
		t.Fatal("expected error for invalid json")
	}
}

func TestMarshalResultEnvelope_Success(t *testing.T) {
	t.Parallel()
	env := &ResultEnvelope{
		CorrelationID: "corr-r1",
		CommandName:   "cmd-r1",
		ResultName:    "result-r1",
		CreatedAt:     time.Now().UTC(),
		Payload:       json.RawMessage(`{"ok":true}`),
		Error:         nil,
	}
	data, err := marshalResultEnvelope(env)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty data")
	}
}

func TestUnmarshalResultEnvelope_Success(t *testing.T) {
	t.Parallel()
	errStr := "some error"
	original := &ResultEnvelope{
		CorrelationID: "corr-r2",
		CommandName:   "cmd-r2",
		CreatedAt:     time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		Error:         &errStr,
	}
	data, _ := marshalResultEnvelope(original)
	env, err := unmarshalResultEnvelope(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.CorrelationID != original.CorrelationID {
		t.Errorf("expected corr %q, got %q", original.CorrelationID, env.CorrelationID)
	}
	if env.Error == nil || *env.Error != errStr {
		t.Errorf("expected error %q, got %v", errStr, env.Error)
	}
}

func TestUnmarshalResultEnvelope_InvalidJSON(t *testing.T) {
	t.Parallel()
	_, err := unmarshalResultEnvelope([]byte("{broken"))
	if err == nil {
		t.Fatal("expected error for invalid json")
	}
}

func TestErrorToPtr_Nil(t *testing.T) {
	t.Parallel()
	ptr := errorToPtr(nil)
	if ptr != nil {
		t.Errorf("expected nil, got %v", ptr)
	}
}

func TestErrorToPtr_NonNil(t *testing.T) {
	t.Parallel()
	err := errors.New("test error")
	ptr := errorToPtr(err)
	if ptr == nil {
		t.Fatal("expected non-nil pointer")
	}
	if *ptr != "test error" {
		t.Errorf("expected %q, got %q", "test error", *ptr)
	}
}
