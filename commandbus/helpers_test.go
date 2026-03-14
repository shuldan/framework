package commandbus

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/shuldan/commands"
)

// --- stub command ---

type stubCommand struct {
	Name    string `json:"name"`
	Payload string `json:"payload"`
	IdemKey string `json:"idem_key,omitempty"`
}

func (c *stubCommand) CommandName() string    { return c.Name }
func (c *stubCommand) IdempotencyKey() string { return c.IdemKey }

// --- fail marshal command ---

type failMarshalCommand struct{}

func (c *failMarshalCommand) CommandName() string    { return "fail-marshal" }
func (c *failMarshalCommand) IdempotencyKey() string { return "fail-key" }
func (c *failMarshalCommand) MarshalJSON() ([]byte, error) {
	return nil, errors.New("marshal error")
}

// --- stub result ---

type stubResult struct {
	commands.BaseResult
	Value string `json:"value"`
}

// --- stub deserializers ---

func stubDeserializer(
	payload []byte, _ *CommandEnvelope,
) (commands.Command, error) {
	var cmd stubCommand
	if err := json.Unmarshal(payload, &cmd); err != nil {
		return nil, err
	}
	return &cmd, nil
}

func failDeserializer(
	_ []byte, _ *CommandEnvelope,
) (commands.Command, error) {
	return nil, errors.New("deserialize error")
}

func stubResultDeserializer(
	payload []byte, _ *ResultEnvelope,
) (commands.Result, error) {
	var r stubResult
	if err := json.Unmarshal(payload, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func failResultDeserializer(
	_ []byte, _ *ResultEnvelope,
) (commands.Result, error) {
	return nil, errors.New("result deserialize error")
}

// --- stub handlers (CommandHandler interface via CommandHandlerFunc) ---

var stubHandler CommandHandler = CommandHandlerFunc(
	func(_ context.Context, _ commands.Command) (commands.Result, error) {
		return &stubResult{
			BaseResult: commands.BaseResult{Name: "stub-result"},
			Value:      "ok",
		}, nil
	},
)

var failHandler CommandHandler = CommandHandlerFunc(
	func(_ context.Context, _ commands.Command) (commands.Result, error) {
		return nil, errors.New("handler error")
	},
)

// --- struct command handler (implements CommandHandler) ---

type structCommandHandler struct {
	mu         sync.Mutex
	value      string
	shouldFail bool
	called     bool
}

func (h *structCommandHandler) Handle(
	_ context.Context, _ commands.Command,
) (commands.Result, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.called = true
	if h.shouldFail {
		return nil, errors.New("struct handler error")
	}
	return &stubResult{
		BaseResult: commands.BaseResult{Name: "struct-result"},
		Value:      h.value,
	}, nil
}

func (h *structCommandHandler) wasCalled() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.called
}

// --- struct result callback (implements ResultCallback) ---

type structResultCallback struct {
	mu         sync.Mutex
	called     bool
	gotResult  commands.Result
	gotErr     error
	shouldFail bool
}

func (c *structResultCallback) OnResult(
	_ context.Context, result commands.Result, err error,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.called = true
	c.gotResult = result
	c.gotErr = err
	if c.shouldFail {
		return errors.New("struct callback error")
	}
	return nil
}

func (c *structResultCallback) wasCalled() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.called
}

func (c *structResultCallback) result() commands.Result {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.gotResult
}

func (c *structResultCallback) err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.gotErr
}

// --- stub broker ---

type stubBroker struct {
	mu       sync.Mutex
	messages map[string][][]byte
	prodErr  error
	consErr  error
}

func newStubBroker() *stubBroker {
	return &stubBroker{
		messages: make(map[string][][]byte),
	}
}

func (b *stubBroker) Produce(_ context.Context, topic string, data []byte) error {
	if b.prodErr != nil {
		return b.prodErr
	}
	b.mu.Lock()
	b.messages[topic] = append(b.messages[topic], data)
	b.mu.Unlock()
	return nil
}

func (b *stubBroker) Consume(ctx context.Context, _ string, _ func([]byte) error) error {
	if b.consErr != nil {
		return b.consErr
	}
	<-ctx.Done()
	return ctx.Err()
}

func (b *stubBroker) Ping(_ context.Context) error { return nil }

func (b *stubBroker) Close() error { return nil }

func (b *stubBroker) getMessages(topic string) [][]byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.messages[topic]
}

// --- callback broker ---

type callbackBroker struct {
	consumeFn func(ctx context.Context, topic string, handler func([]byte) error) error
	produceFn func(ctx context.Context, topic string, data []byte) error
}

func newCallbackBroker() *callbackBroker {
	return &callbackBroker{}
}

func (b *callbackBroker) Produce(ctx context.Context, topic string, data []byte) error {
	if b.produceFn != nil {
		return b.produceFn(ctx, topic, data)
	}
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

func (b *callbackBroker) Close() error { return nil }

// --- stub idempotency store ---

type stubIdempotencyStore struct {
	mu       sync.Mutex
	keys     map[string]bool
	existErr error
	markErr  error
}

func newStubIdempotencyStore() *stubIdempotencyStore {
	return &stubIdempotencyStore{keys: make(map[string]bool)}
}

func (s *stubIdempotencyStore) Exists(_ context.Context, key string) (bool, error) {
	if s.existErr != nil {
		return false, s.existErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.keys[key], nil
}

func (s *stubIdempotencyStore) Mark(_ context.Context, key string, _ time.Duration) error {
	if s.markErr != nil {
		return s.markErr
	}
	s.mu.Lock()
	s.keys[key] = true
	s.mu.Unlock()
	return nil
}

// --- recording logger ---

type logEntry struct {
	level string
	msg   string
}

type recordingLogger struct {
	mu      sync.Mutex
	entries []logEntry
}

func newRecordingLogger() *recordingLogger {
	return &recordingLogger{}
}

func (l *recordingLogger) Info(msg string, _ ...any) {
	l.mu.Lock()
	l.entries = append(l.entries, logEntry{level: "info", msg: msg})
	l.mu.Unlock()
}

func (l *recordingLogger) Warn(msg string, _ ...any) {
	l.mu.Lock()
	l.entries = append(l.entries, logEntry{level: "warn", msg: msg})
	l.mu.Unlock()
}

func (l *recordingLogger) Error(msg string, _ ...any) {
	l.mu.Lock()
	l.entries = append(l.entries, logEntry{level: "error", msg: msg})
	l.mu.Unlock()
}

func (l *recordingLogger) Debug(msg string, _ ...any) {
	l.mu.Lock()
	l.entries = append(l.entries, logEntry{level: "debug", msg: msg})
	l.mu.Unlock()
}

func (l *recordingLogger) hasInfo(msg string) bool  { return l.has("info", msg) }
func (l *recordingLogger) hasWarn(msg string) bool  { return l.has("warn", msg) }
func (l *recordingLogger) hasError(msg string) bool { return l.has("error", msg) }

func (l *recordingLogger) has(level, msg string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, e := range l.entries {
		if e.level == level && e.msg == msg {
			return true
		}
	}
	return false
}

// --- envelope builders ---

func makeCommandEnvelopeBytes(
	cmdName, idemKey, replyTo string, timeout time.Duration,
) []byte {
	cmd := &stubCommand{Name: cmdName, Payload: "test"}
	payload, _ := json.Marshal(cmd)
	env := &CommandEnvelope{
		IdempotencyKey: idemKey,
		CommandName:    cmdName,
		ReplyTo:        replyTo,
		CorrelationID:  "corr-" + idemKey,
		CreatedAt:      time.Now().UTC(),
		Timeout:        timeout,
		Payload:        payload,
	}
	data, _ := marshalCommandEnvelope(env)
	return data
}

func makeExpiredEnvelopeBytes(
	cmdName, idemKey, replyTo string,
) []byte {
	cmd := &stubCommand{Name: cmdName, Payload: "test"}
	payload, _ := json.Marshal(cmd)
	env := &CommandEnvelope{
		IdempotencyKey: idemKey,
		CommandName:    cmdName,
		ReplyTo:        replyTo,
		CorrelationID:  "corr-" + idemKey,
		CreatedAt:      time.Now().UTC().Add(-2 * time.Hour),
		Timeout:        1 * time.Hour,
		Payload:        payload,
	}
	data, _ := marshalCommandEnvelope(env)
	return data
}

func makeResultEnvelopeBytes(
	cmdName, corrID string,
	errPtr *string,
	result commands.Result,
) []byte {
	env := &ResultEnvelope{
		CorrelationID: corrID,
		CommandName:   cmdName,
		CreatedAt:     time.Now().UTC(),
		Error:         errPtr,
	}
	if result != nil {
		env.ResultName = result.ResultName()
		payload, _ := json.Marshal(result)
		env.Payload = payload
	}
	data, _ := marshalResultEnvelope(env)
	return data
}

// --- helpers ---

func mustUnmarshalCommandEnvelope(data []byte) *CommandEnvelope {
	env, err := unmarshalCommandEnvelope(data)
	if err != nil {
		panic("mustUnmarshalCommandEnvelope: " + err.Error())
	}
	return env
}

func errContains(err error, substr string) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), substr)
}
