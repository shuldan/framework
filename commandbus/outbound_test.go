package commandbus

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/shuldan/commands"
)

func TestNewCommandSender_Defaults(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	s := NewCommandSender(br, nil)
	if s.timeout != defaultTimeout {
		t.Errorf("expected timeout %v, got %v", defaultTimeout, s.timeout)
	}
	if s.replyTo != "" {
		t.Errorf("expected empty replyTo, got %q", s.replyTo)
	}
}

func TestNewCommandSender_WithOptions(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	s := NewCommandSender(br, nil,
		WithReplyTo("my-svc"),
		WithSender("sender-svc"),
		WithDefaultTimeout(10*time.Second),
	)
	if s.replyTo != "my-svc" {
		t.Errorf("expected replyTo %q, got %q", "my-svc", s.replyTo)
	}
	if s.sender != "sender-svc" {
		t.Errorf("expected sender %q, got %q", "sender-svc", s.sender)
	}
	if s.timeout != 10*time.Second {
		t.Errorf("expected timeout %v, got %v", 10*time.Second, s.timeout)
	}
}

func TestCommandSender_Forward(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	log := newRecordingLogger()
	s := NewCommandSender(br, log)
	s.Forward("test.command")
	if _, ok := s.routes["test.command"]; !ok {
		t.Fatal("route not registered")
	}
	if !log.hasInfo("command sender: route registered") {
		t.Error("expected info log for route registration")
	}
}

func TestCommandSender_Send_NilCommand(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	s := NewCommandSender(br, nil)
	err := s.Send(context.Background(), nil)
	if !errors.Is(err, commands.ErrNilCommand) {
		t.Errorf("expected ErrNilCommand, got %v", err)
	}
}

func TestCommandSender_Send_NoRoute(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	s := NewCommandSender(br, nil)
	cmd := &stubCommand{Name: "unregistered"}
	err := s.Send(context.Background(), cmd)
	if !errors.Is(err, commands.ErrHandlerNotFound) {
		t.Errorf("expected ErrHandlerNotFound, got %v", err)
	}
}

func TestCommandSender_Send_Success(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	log := newRecordingLogger()
	s := NewCommandSender(br, log, WithReplyTo("my-svc"))
	s.Forward("test.cmd")
	cmd := &stubCommand{Name: "test.cmd", Payload: "data"}
	cmd.IdemKey = "idem-1"
	err := s.Send(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msgs := br.getMessages("commands.test.cmd")
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	env := mustUnmarshalCommandEnvelope(msgs[0])
	if env.IdempotencyKey != "idem-1" {
		t.Errorf("expected key %q, got %q", "idem-1", env.IdempotencyKey)
	}
	if env.ReplyTo != "my-svc" {
		t.Errorf("expected replyTo %q, got %q", "my-svc", env.ReplyTo)
	}
}

func TestCommandSender_Send_WithSendOptions(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	s := NewCommandSender(br, nil, WithReplyTo("svc"))
	s.Forward("test.cmd")
	cmd := &stubCommand{Name: "test.cmd"}
	headers := map[string]string{"x-trace": "abc"}
	err := s.Send(context.Background(), cmd,
		WithTimeout(99*time.Second),
		WithHeaders(headers),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msgs := br.getMessages("commands.test.cmd")
	env := mustUnmarshalCommandEnvelope(msgs[0])
	if env.Timeout != 99*time.Second {
		t.Errorf("expected timeout %v, got %v", 99*time.Second, env.Timeout)
	}
	if env.Headers["x-trace"] != "abc" {
		t.Errorf("expected header x-trace=abc, got %v", env.Headers)
	}
}

func TestCommandSender_Send_WithoutReply(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	s := NewCommandSender(br, nil, WithReplyTo("svc"))
	s.Forward("test.cmd")
	cmd := &stubCommand{Name: "test.cmd"}
	err := s.Send(context.Background(), cmd, WithoutReply())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msgs := br.getMessages("commands.test.cmd")
	env := mustUnmarshalCommandEnvelope(msgs[0])
	if env.ReplyTo != "" {
		t.Errorf("expected empty replyTo, got %q", env.ReplyTo)
	}
}

func TestCommandSender_Send_BrokerError(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	br.prodErr = errors.New("broker down")
	s := NewCommandSender(br, nil)
	s.Forward("test.cmd")
	cmd := &stubCommand{Name: "test.cmd"}
	err := s.Send(context.Background(), cmd)
	if !errContains(err, "broker down") {
		t.Errorf("expected broker error, got %v", err)
	}
}

func TestCommandSender_Send_MarshalError(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	s := NewCommandSender(br, nil)
	s.Forward("fail-marshal")
	cmd := &failMarshalCommand{}
	err := s.Send(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected marshal error")
	}
	if !errContains(err, "marshal") {
		t.Errorf("expected marshal in error, got %v", err)
	}
}

func TestCommandSender_Send_EmptyIdempotencyKey(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	s := NewCommandSender(br, nil)
	s.Forward("test.cmd")
	cmd := &stubCommand{Name: "test.cmd"}
	err := s.Send(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msgs := br.getMessages("commands.test.cmd")
	env := mustUnmarshalCommandEnvelope(msgs[0])
	if env.IdempotencyKey == "" {
		t.Error("expected auto-generated idempotency key")
	}
}

type unmarshalableCommand struct {
	Name string `json:"name"`
	Ch   chan int
}

func (u *unmarshalableCommand) CommandName() string    { return u.Name }
func (u *unmarshalableCommand) IdempotencyKey() string { return "" }

func TestCommandSender_Send_MarshalEnvelopeError(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	s := NewCommandSender(br, nil)
	s.Forward("fail-marshal")
	cmd := &failMarshalCommand{}
	err := s.Send(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errContains(err, "build envelope") || !errContains(err, "marshal") {
		t.Errorf("expected marshal error in build envelope, got %v", err)
	}
}

func TestCommandSender_BuildEnvelope_EmptyIdempotencyKey(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	s := NewCommandSender(br, nil)
	s.Forward("test.cmd")
	cmd := &stubCommand{Name: "test.cmd", Payload: "data"}
	err := s.Send(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msgs := br.getMessages("commands.test.cmd")
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	env := mustUnmarshalCommandEnvelope(msgs[0])
	if env.IdempotencyKey == "" {
		t.Error("expected auto-generated UUID idempotency key")
	}
	if len(env.IdempotencyKey) < 32 {
		t.Errorf("expected UUID-like key, got %q", env.IdempotencyKey)
	}
}

func TestCommandSender_BuildEnvelope_NonEmptyIdempotencyKey(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	s := NewCommandSender(br, nil)
	s.Forward("test.cmd")
	cmd := &stubCommand{Name: "test.cmd", Payload: "data"}
	cmd.IdemKey = "my-custom-key"
	err := s.Send(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msgs := br.getMessages("commands.test.cmd")
	env := mustUnmarshalCommandEnvelope(msgs[0])
	if env.IdempotencyKey != "my-custom-key" {
		t.Errorf("expected %q, got %q", "my-custom-key", env.IdempotencyKey)
	}
}

func TestCommandSender_Send_VerifyEnvelopeFields(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	s := NewCommandSender(br, nil,
		WithReplyTo("rpc-svc"),
		WithSender("api-gw"),
	)
	s.Forward("test.cmd")
	cmd := &stubCommand{Name: "test.cmd", Payload: "verify"}
	cmd.IdemKey = "verify-key"
	err := s.Send(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msgs := br.getMessages("commands.test.cmd")
	env := mustUnmarshalCommandEnvelope(msgs[0])
	if env.Sender != "api-gw" {
		t.Errorf("expected sender %q, got %q", "api-gw", env.Sender)
	}
	if env.CommandName != "test.cmd" {
		t.Errorf("expected command %q, got %q", "test.cmd", env.CommandName)
	}
	if env.CorrelationID == "" {
		t.Error("expected non-empty correlation ID")
	}
	if env.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
	var payload stubCommand
	if unmErr := json.Unmarshal(env.Payload, &payload); unmErr != nil {
		t.Fatalf("unmarshal payload: %v", unmErr)
	}
	if payload.Payload != "verify" {
		t.Errorf("expected payload %q, got %q", "verify", payload.Payload)
	}
}

func TestCommandSender_Send_MarshalCommandEnvelopeStruct(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	br.prodErr = errors.New("should not reach")
	s := NewCommandSender(br, nil)
	s.Forward("fail-marshal")
	cmd := &failMarshalCommand{}
	err := s.Send(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error")
	}
	if errContains(err, "should not reach") {
		t.Error("should not have reached broker.Produce")
	}
}
