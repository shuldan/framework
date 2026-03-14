package commandbus

import (
	"context"
	"encoding/json"
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

func TestCommandReceiver_Registrations_RunCallsBrokerConsume(t *testing.T) {
	t.Parallel()
	cb := newCallbackBroker()
	var consumedTopic string
	var handlerCalled bool
	cb.consumeFn = func(ctx context.Context, topic string, handler func([]byte) error) error {
		consumedTopic = topic
		data := makeCommandEnvelopeBytes("run.cmd", "run-key", "", 0)
		handlerCalled = true
		return handler(data)
	}
	r := NewCommandReceiver(cb, nil)
	_ = r.Handle("run.cmd", stubDeserializer, stubHandler)
	regs := r.Registrations()
	if len(regs) != 1 {
		t.Fatalf("expected 1 registration, got %d", len(regs))
	}
	err := regs[0].Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if consumedTopic != "commands.run.cmd" {
		t.Errorf("expected topic %q, got %q", "commands.run.cmd", consumedTopic)
	}
	if !handlerCalled {
		t.Error("expected handler to be called via broker.Consume")
	}
}

func TestCommandReceiver_Registrations_RunHandlerError(t *testing.T) {
	t.Parallel()
	cb := newCallbackBroker()
	cb.consumeFn = func(_ context.Context, _ string, handler func([]byte) error) error {
		data := makeCommandEnvelopeBytes("run.cmd", "run-key2", "", 0)
		return handler(data)
	}
	r := NewCommandReceiver(cb, nil)
	_ = r.Handle("run.cmd", stubDeserializer, failHandler)
	regs := r.Registrations()
	err := regs[0].Run(context.Background())
	if err == nil {
		t.Fatal("expected error from failed handler")
	}
	if !errContains(err, "handler error") {
		t.Errorf("expected handler error, got %v", err)
	}
}

func TestCommandReceiver_Registrations_RunBrokerConsumeError(t *testing.T) {
	t.Parallel()
	cb := newCallbackBroker()
	cb.consumeFn = func(_ context.Context, _ string, _ func([]byte) error) error {
		return errors.New("consume failed")
	}
	r := NewCommandReceiver(cb, nil)
	_ = r.Handle("run.cmd", stubDeserializer, stubHandler)
	regs := r.Registrations()
	err := regs[0].Run(context.Background())
	if err == nil {
		t.Fatal("expected consume error")
	}
	if !errContains(err, "consume failed") {
		t.Errorf("expected 'consume failed', got %v", err)
	}
}

type failMarshalResult struct {
	commands.BaseResult
}

func (f *failMarshalResult) MarshalJSON() ([]byte, error) {
	return nil, errors.New("result marshal error")
}

func TestCommandReceiver_SendResult_MarshalResultPayloadError(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	r := NewCommandReceiver(br, nil)
	cmdEnv := &CommandEnvelope{
		CorrelationID: "c-marshal",
		CommandName:   "test.cmd",
		ReplyTo:       "svc",
	}
	badResult := &failMarshalResult{
		BaseResult: commands.BaseResult{Name: "bad"},
	}
	err := r.sendResult(context.Background(), cmdEnv, badResult, nil)
	if err == nil {
		t.Fatal("expected marshal result payload error")
	}
	if !errContains(err, "marshal result payload") {
		t.Errorf("expected 'marshal result payload' in error, got %v", err)
	}
}

func TestCommandReceiver_SendResult_MarshalEnvelopeError(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	r := NewCommandReceiver(br, nil)
	cmdEnv := &CommandEnvelope{
		CorrelationID: "c-env-marshal",
		CommandName:   "test.cmd",
		ReplyTo:       "svc",
	}
	goodResult := &stubResult{
		BaseResult: commands.BaseResult{Name: "ok"},
		Value:      "val",
	}
	err := r.sendResult(context.Background(), cmdEnv, goodResult, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msgs := br.getMessages("replies.svc")
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	var env ResultEnvelope
	if unmErr := json.Unmarshal(msgs[0], &env); unmErr != nil {
		t.Fatalf("unmarshal result envelope: %v", unmErr)
	}
	if env.CorrelationID != "c-env-marshal" {
		t.Errorf("expected correlation %q, got %q", "c-env-marshal", env.CorrelationID)
	}
}

func TestCommandReceiver_HandleMessage_SuccessResult_VerifyEnvelope(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	r := NewCommandReceiver(br, nil)
	_ = r.Handle("test.cmd", stubDeserializer, func(_ context.Context, _ commands.Command) (commands.Result, error) {
		return &stubResult{
			BaseResult: commands.BaseResult{Name: "verified-result"},
			Value:      "data",
		}, nil
	})
	data := makeCommandEnvelopeBytes("test.cmd", "verify-key", "reply-svc", 0)
	err := r.handleMessage(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msgs := br.getMessages("replies.reply-svc")
	if len(msgs) != 1 {
		t.Fatalf("expected 1 reply, got %d", len(msgs))
	}
	var env ResultEnvelope
	_ = json.Unmarshal(msgs[0], &env)
	if env.ResultName != "verified-result" {
		t.Errorf("expected result name %q, got %q", "verified-result", env.ResultName)
	}
	if env.Error != nil {
		t.Errorf("expected nil error, got %v", *env.Error)
	}
	if len(env.Payload) == 0 {
		t.Error("expected non-empty payload")
	}
}

func TestCommandReceiver_HandleMessage_HandlerReturnsFailMarshalResult(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	log := newRecordingLogger()
	r := NewCommandReceiver(br, log)
	_ = r.Handle("test.cmd", stubDeserializer, func(_ context.Context, _ commands.Command) (commands.Result, error) {
		return &failMarshalResult{BaseResult: commands.BaseResult{Name: "bad"}}, nil
	})
	data := makeCommandEnvelopeBytes("test.cmd", "bad-marshal-key", "reply-svc", 0)
	err := r.handleMessage(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !log.hasError("command receiver: send result failed") {
		t.Error("expected send result failed log")
	}
}

func TestCommandReceiver_Registrations_ContextCancel(t *testing.T) {
	t.Parallel()
	cb := newCallbackBroker()
	cb.consumeFn = func(ctx context.Context, _ string, _ func([]byte) error) error {
		<-ctx.Done()
		return ctx.Err()
	}
	r := NewCommandReceiver(cb, nil)
	_ = r.Handle("ctx.cmd", stubDeserializer, stubHandler)
	regs := r.Registrations()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := regs[0].Run(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
}
