package commandbus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/shuldan/commands"
)

type stubCommand struct {
	commands.BaseCommand
	Name    string `json:"name"`
	Payload string `json:"payload"`
}

func (s *stubCommand) CommandName() string {
	return s.Name
}

type stubResult struct {
	commands.BaseResult
	Value string `json:"value"`
}

type failMarshalCommand struct {
	commands.BaseCommand
}

func (f *failMarshalCommand) CommandName() string { return "fail-marshal" }
func (f *failMarshalCommand) MarshalJSON() ([]byte, error) {
	return nil, errors.New("marshal error")
}

type stubIdempotencyStore struct {
	mu       sync.Mutex
	entries  map[string]bool
	existErr error
	markErr  error
}

func newStubIdempotencyStore() *stubIdempotencyStore {
	return &stubIdempotencyStore{entries: make(map[string]bool)}
}

func (s *stubIdempotencyStore) Exists(_ context.Context, key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.existErr != nil {
		return false, s.existErr
	}
	return s.entries[key], nil
}

func (s *stubIdempotencyStore) Mark(_ context.Context, key string, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.markErr != nil {
		return s.markErr
	}
	s.entries[key] = true
	return nil
}

type stubBroker struct {
	mu       sync.Mutex
	messages map[string][][]byte
	prodErr  error
	consErr  error
	closed   bool
}

func newStubBroker() *stubBroker {
	return &stubBroker{messages: make(map[string][][]byte)}
}

func (b *stubBroker) Produce(_ context.Context, topic string, data []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.prodErr != nil {
		return b.prodErr
	}
	b.messages[topic] = append(b.messages[topic], data)
	return nil
}

func (b *stubBroker) Consume(ctx context.Context, topic string, handler func([]byte) error) error {
	b.mu.Lock()
	if b.consErr != nil {
		err := b.consErr
		b.mu.Unlock()
		return err
	}
	b.mu.Unlock()
	<-ctx.Done()
	return ctx.Err()
}

func (b *stubBroker) Ping(_ context.Context) error { return nil }
func (b *stubBroker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	return nil
}

func (b *stubBroker) getMessages(topic string) [][]byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.messages[topic]
}

type callbackBroker struct {
	mu        sync.Mutex
	produced  map[string][][]byte
	consumeFn func(ctx context.Context, topic string, handler func([]byte) error) error
	prodErr   error
}

func newCallbackBroker() *callbackBroker {
	return &callbackBroker{produced: make(map[string][][]byte)}
}

func (b *callbackBroker) Produce(_ context.Context, topic string, data []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.prodErr != nil {
		return b.prodErr
	}
	b.produced[topic] = append(b.produced[topic], data)
	return nil
}

func (b *callbackBroker) Consume(ctx context.Context, topic string, handler func([]byte) error) error {
	if b.consumeFn != nil {
		return b.consumeFn(ctx, topic, handler)
	}
	<-ctx.Done()
	return ctx.Err()
}

func (b *callbackBroker) Ping(_ context.Context) error { return nil }
func (b *callbackBroker) Close() error                 { return nil }

func (b *callbackBroker) getProduced(topic string) [][]byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.produced[topic]
}

type recordingLogger struct {
	mu     sync.Mutex
	infos  []string
	warns  []string
	errs   []string
	debugs []string
}

func newRecordingLogger() *recordingLogger {
	return &recordingLogger{}
}

func (l *recordingLogger) Info(msg string, _ ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.infos = append(l.infos, msg)
}

func (l *recordingLogger) Warn(msg string, _ ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.warns = append(l.warns, msg)
}

func (l *recordingLogger) Error(msg string, _ ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errs = append(l.errs, msg)
}

func (l *recordingLogger) Debug(msg string, _ ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.debugs = append(l.debugs, msg)
}

func (l *recordingLogger) hasError(substr string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, e := range l.errs {
		if contains(e, substr) {
			return true
		}
	}
	return false
}

func (l *recordingLogger) hasWarn(substr string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, w := range l.warns {
		if contains(w, substr) {
			return true
		}
	}
	return false
}

func (l *recordingLogger) hasInfo(substr string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, i := range l.infos {
		if contains(i, substr) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func makeCommandEnvelopeBytes(name, key, replyTo string, timeout time.Duration) []byte {
	cmd := &stubCommand{Name: name, Payload: "test-payload"}
	cmd.IdemKey = key
	payload, _ := json.Marshal(cmd)
	env := &CommandEnvelope{
		IdempotencyKey: key,
		CommandName:    name,
		ReplyTo:        replyTo,
		CorrelationID:  "corr-123",
		CreatedAt:      time.Now().UTC(),
		Timeout:        timeout,
		Payload:        payload,
	}
	data, _ := marshalCommandEnvelope(env)
	return data
}

func makeExpiredEnvelopeBytes(name, key, replyTo string) []byte {
	cmd := &stubCommand{Name: name, Payload: "test-payload"}
	cmd.IdemKey = key
	payload, _ := json.Marshal(cmd)
	env := &CommandEnvelope{
		IdempotencyKey: key,
		CommandName:    name,
		ReplyTo:        replyTo,
		CorrelationID:  "corr-expired",
		CreatedAt:      time.Now().UTC().Add(-2 * time.Hour),
		Timeout:        1 * time.Second,
		Payload:        payload,
	}
	data, _ := marshalCommandEnvelope(env)
	return data
}

func makeResultEnvelopeBytes(cmdName, corrID string, errMsg *string, result commands.Result) []byte {
	env := &ResultEnvelope{
		CorrelationID: corrID,
		CommandName:   cmdName,
		CreatedAt:     time.Now().UTC(),
		Error:         errMsg,
	}
	if result != nil {
		env.ResultName = result.ResultName()
		payload, _ := json.Marshal(result)
		env.Payload = payload
	}
	data, _ := marshalResultEnvelope(env)
	return data
}

func stubDeserializer(payload []byte, _ *CommandEnvelope) (commands.Command, error) {
	var cmd stubCommand
	if err := json.Unmarshal(payload, &cmd); err != nil {
		return nil, err
	}
	return &cmd, nil
}

func failDeserializer(_ []byte, _ *CommandEnvelope) (commands.Command, error) {
	return nil, errors.New("deserialize failed")
}

func stubHandler(_ context.Context, _ commands.Command) (commands.Result, error) {
	return &stubResult{
		BaseResult: commands.BaseResult{Name: "stub-result"},
		Value:      "ok",
	}, nil
}

func failHandler(_ context.Context, _ commands.Command) (commands.Result, error) {
	return nil, errors.New("handler error")
}

func stubResultDeserializer(payload []byte, _ *ResultEnvelope) (commands.Result, error) {
	var r stubResult
	if err := json.Unmarshal(payload, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func failResultDeserializer(_ []byte, _ *ResultEnvelope) (commands.Result, error) {
	return nil, errors.New("result deserialize failed")
}

func strPtr(s string) *string { return &s }

func errContains(err error, substr string) bool {
	if err == nil {
		return false
	}
	return contains(err.Error(), substr)
}

func mustUnmarshalCommandEnvelope(data []byte) *CommandEnvelope {
	env, err := unmarshalCommandEnvelope(data)
	if err != nil {
		panic(fmt.Sprintf("unmarshal command envelope: %v", err))
	}
	return env
}
