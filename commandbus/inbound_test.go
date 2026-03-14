package commandbus

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shuldan/commands"
)

func TestNewCommandReceiver_Defaults(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	r := NewCommandReceiver(br, nil)
	if r.idemTTL != defaultIdemTTL {
		t.Errorf("expected idemTTL %v, got %v", defaultIdemTTL, r.idemTTL)
	}
	if r.idemStore == nil {
		t.Fatal("expected non-nil idempotency store")
	}
}

func TestNewCommandReceiver_WithOptions(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	store := newStubIdempotencyStore()
	r := NewCommandReceiver(br, nil,
		WithIdempotencyStore(store),
		WithIdempotencyTTL(1*time.Hour),
	)
	if r.idemTTL != 1*time.Hour {
		t.Errorf("expected idemTTL 1h, got %v", r.idemTTL)
	}
}

func TestCommandReceiver_Handle_Success(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	log := newRecordingLogger()
	r := NewCommandReceiver(br, log)
	err := r.Handle("test.cmd", stubDeserializer, stubHandler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !log.hasInfo("command receiver: handler registered") {
		t.Error("expected registration log")
	}
}

func TestCommandReceiver_Handle_Duplicate(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	r := NewCommandReceiver(br, nil)
	_ = r.Handle("test.cmd", stubDeserializer, stubHandler)
	err := r.Handle("test.cmd", stubDeserializer, stubHandler)
	if !errors.Is(err, commands.ErrHandlerExists) {
		t.Errorf("expected ErrHandlerExists, got %v", err)
	}
}

func TestCommandReceiver_Handle_WithCommandIdempotencyTTL(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	r := NewCommandReceiver(br, nil)
	err := r.Handle("test.cmd", stubDeserializer, stubHandler,
		WithCommandIdempotencyTTL(5*time.Minute),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r.mu.RLock()
	entry := r.entries["test.cmd"]
	r.mu.RUnlock()
	if entry.idemTTL != 5*time.Minute {
		t.Errorf("expected idemTTL 5m, got %v", entry.idemTTL)
	}
}

func TestCommandReceiver_Registrations(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	r := NewCommandReceiver(br, nil)
	_ = r.Handle("cmd-a", stubDeserializer, stubHandler)
	_ = r.Handle("cmd-b", stubDeserializer, stubHandler)
	regs := r.Registrations()
	if len(regs) != 2 {
		t.Fatalf("expected 2 registrations, got %d", len(regs))
	}
}

func TestCommandReceiver_HandleMessage_InvalidJSON(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	log := newRecordingLogger()
	r := NewCommandReceiver(br, log)
	err := r.handleMessage(context.Background(), []byte("bad-json"))
	if err != nil {
		t.Fatalf("expected nil error (message dropped), got %v", err)
	}
	if !log.hasError("command receiver: unmarshal failed") {
		t.Error("expected unmarshal error log")
	}
}

func TestCommandReceiver_HandleMessage_Expired_NoReply(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	log := newRecordingLogger()
	r := NewCommandReceiver(br, log)
	_ = r.Handle("test.cmd", stubDeserializer, stubHandler)
	data := makeExpiredEnvelopeBytes("test.cmd", "key-exp", "")
	err := r.handleMessage(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !log.hasWarn("command receiver: command expired") {
		t.Error("expected expired warning log")
	}
}

func TestCommandReceiver_HandleMessage_Expired_WithReply(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	log := newRecordingLogger()
	r := NewCommandReceiver(br, log)
	_ = r.Handle("test.cmd", stubDeserializer, stubHandler)
	data := makeExpiredEnvelopeBytes("test.cmd", "key-exp2", "reply-svc")
	err := r.handleMessage(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msgs := br.getMessages("replies.reply-svc")
	if len(msgs) == 0 {
		t.Fatal("expected reply message for expired command")
	}
}

func TestCommandReceiver_HandleMessage_Duplicate(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	log := newRecordingLogger()
	store := newStubIdempotencyStore()
	r := NewCommandReceiver(br, log, WithIdempotencyStore(store))
	_ = r.Handle("test.cmd", stubDeserializer, stubHandler)
	_ = store.Mark(context.Background(), "dup-key", time.Hour)
	data := makeCommandEnvelopeBytes("test.cmd", "dup-key", "", 0)
	err := r.handleMessage(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCommandReceiver_HandleMessage_NoHandler(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	log := newRecordingLogger()
	r := NewCommandReceiver(br, log)
	data := makeCommandEnvelopeBytes("unknown.cmd", "key-1", "", 0)
	err := r.handleMessage(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !log.hasError("command receiver: no handler") {
		t.Error("expected no handler error log")
	}
}

func TestCommandReceiver_HandleMessage_DeserializeFail(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	log := newRecordingLogger()
	r := NewCommandReceiver(br, log)
	_ = r.Handle("test.cmd", failDeserializer, stubHandler)
	data := makeCommandEnvelopeBytes("test.cmd", "key-deser", "", 0)
	err := r.handleMessage(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !log.hasError("command receiver: deserialize failed") {
		t.Error("expected deserialize error log")
	}
}

func TestCommandReceiver_HandleMessage_HandlerError_NoReply(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	log := newRecordingLogger()
	r := NewCommandReceiver(br, log)
	_ = r.Handle("test.cmd", stubDeserializer, failHandler)
	data := makeCommandEnvelopeBytes("test.cmd", "key-fail", "", 0)
	err := r.handleMessage(context.Background(), data)
	if err == nil {
		t.Fatal("expected error from handler")
	}
	if !errContains(err, "handler error") {
		t.Errorf("expected handler error in message, got %v", err)
	}
}

func TestCommandReceiver_HandleMessage_HandlerError_WithReply(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	log := newRecordingLogger()
	r := NewCommandReceiver(br, log)
	_ = r.Handle("test.cmd", stubDeserializer, failHandler)
	data := makeCommandEnvelopeBytes("test.cmd", "key-fail-reply", "reply-svc", 0)
	err := r.handleMessage(context.Background(), data)
	if err == nil {
		t.Fatal("expected error")
	}
	msgs := br.getMessages("replies.reply-svc")
	if len(msgs) == 0 {
		t.Fatal("expected reply with error")
	}
}

func TestCommandReceiver_HandleMessage_Success_WithReply(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	log := newRecordingLogger()
	r := NewCommandReceiver(br, log)
	_ = r.Handle("test.cmd", stubDeserializer, stubHandler)
	data := makeCommandEnvelopeBytes("test.cmd", "key-ok", "reply-svc", 0)
	err := r.handleMessage(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msgs := br.getMessages("replies.reply-svc")
	if len(msgs) == 0 {
		t.Fatal("expected reply message")
	}
}

func TestCommandReceiver_HandleMessage_Success_NoReply(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	r := NewCommandReceiver(br, nil)
	_ = r.Handle("test.cmd", stubDeserializer, stubHandler)
	data := makeCommandEnvelopeBytes("test.cmd", "key-ok-nr", "", 0)
	err := r.handleMessage(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCommandReceiver_HandleMessage_IdempotencyCheckError(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	log := newRecordingLogger()
	store := newStubIdempotencyStore()
	store.existErr = errors.New("store error")
	r := NewCommandReceiver(br, log, WithIdempotencyStore(store))
	_ = r.Handle("test.cmd", stubDeserializer, stubHandler)
	data := makeCommandEnvelopeBytes("test.cmd", "key-err", "", 0)
	err := r.handleMessage(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !log.hasError("command receiver: idempotency check failed") {
		t.Error("expected idempotency check error log")
	}
}

func TestCommandReceiver_HandleMessage_IdempotencyMarkError(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	log := newRecordingLogger()
	store := newStubIdempotencyStore()
	store.markErr = errors.New("mark error")
	r := NewCommandReceiver(br, log, WithIdempotencyStore(store))
	_ = r.Handle("test.cmd", stubDeserializer, stubHandler)
	data := makeCommandEnvelopeBytes("test.cmd", "key-mark-err", "", 0)
	err := r.handleMessage(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !log.hasError("command receiver: idempotency mark failed") {
		t.Error("expected idempotency mark error log")
	}
}

func TestCommandReceiver_SendResult_BrokerError(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	br.prodErr = errors.New("produce fail")
	log := newRecordingLogger()
	r := NewCommandReceiver(br, log)
	_ = r.Handle("test.cmd", stubDeserializer, stubHandler)
	data := makeCommandEnvelopeBytes("test.cmd", "key-brok", "reply-svc", 0)
	err := r.handleMessage(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !log.hasError("command receiver: send result failed") {
		t.Error("expected send result error log")
	}
}

func TestCommandReceiver_IsExpired_NoTimeout(t *testing.T) {
	t.Parallel()
	r := NewCommandReceiver(newStubBroker(), nil)
	env := &CommandEnvelope{Timeout: 0}
	if r.isExpired(env) {
		t.Error("expected not expired with zero timeout")
	}
}

func TestCommandReceiver_IsExpired_NegativeTimeout(t *testing.T) {
	t.Parallel()
	r := NewCommandReceiver(newStubBroker(), nil)
	env := &CommandEnvelope{Timeout: -1 * time.Second}
	if r.isExpired(env) {
		t.Error("expected not expired with negative timeout")
	}
}

func TestCommandReceiver_IsExpired_NotYetExpired(t *testing.T) {
	t.Parallel()
	r := NewCommandReceiver(newStubBroker(), nil)
	env := &CommandEnvelope{
		CreatedAt: time.Now().UTC(),
		Timeout:   1 * time.Hour,
	}
	if r.isExpired(env) {
		t.Error("expected not expired")
	}
}
