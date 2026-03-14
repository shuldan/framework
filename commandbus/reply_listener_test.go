package commandbus

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/shuldan/commands"
)

func TestNewReplyListener_Defaults(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	rl := NewReplyListener(br, nil)
	if rl.serviceName != "" {
		t.Errorf("expected empty service name, got %q", rl.serviceName)
	}
}

func TestNewReplyListener_WithServiceName(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	rl := NewReplyListener(br, nil, WithListenerServiceName("my-svc"))
	if rl.serviceName != "my-svc" {
		t.Errorf("expected %q, got %q", "my-svc", rl.serviceName)
	}
}

func TestReplyListener_OnResult_WithFunc(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	log := newRecordingLogger()
	rl := NewReplyListener(br, log)
	rl.OnResult("test.cmd", stubResultDeserializer,
		ResultCallbackFunc(func(_ context.Context, _ commands.Result, _ error) error {
			return nil
		}),
	)
	if !log.hasInfo("reply listener: result handler registered") {
		t.Error("expected registration log")
	}
	rl.mu.RLock()
	_, ok := rl.entries["test.cmd"]
	rl.mu.RUnlock()
	if !ok {
		t.Error("expected entry to be registered")
	}
}

func TestReplyListener_OnResult_WithStruct(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	log := newRecordingLogger()
	rl := NewReplyListener(br, log)
	cb := &structResultCallback{}
	rl.OnResult("test.cmd", stubResultDeserializer, cb)
	if !log.hasInfo("reply listener: result handler registered") {
		t.Error("expected registration log")
	}
	rl.mu.RLock()
	_, ok := rl.entries["test.cmd"]
	rl.mu.RUnlock()
	if !ok {
		t.Error("expected entry to be registered")
	}
}

func TestReplyListener_HandleMessage_InvalidJSON(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	log := newRecordingLogger()
	rl := NewReplyListener(br, log)
	err := rl.handleMessage(context.Background(), []byte("bad"))
	if err != nil {
		t.Fatalf("expected nil (message dropped), got %v", err)
	}
	if !log.hasError("reply listener: unmarshal failed") {
		t.Error("expected unmarshal error log")
	}
}

func TestReplyListener_HandleMessage_NoHandler(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	log := newRecordingLogger()
	rl := NewReplyListener(br, log)
	data := makeResultEnvelopeBytes("unknown.cmd", "c1", nil, nil)
	err := rl.handleMessage(context.Background(), data)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if !log.hasError("reply listener: no result handler") {
		t.Error("expected no handler error log")
	}
}

func TestReplyListener_HandleMessage_WithError(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	rl := NewReplyListener(br, nil)
	var gotErr error
	rl.OnResult("test.cmd", stubResultDeserializer,
		ResultCallbackFunc(func(_ context.Context, _ commands.Result, err error) error {
			gotErr = err
			return nil
		}),
	)
	errMsg := "remote error"
	data := makeResultEnvelopeBytes("test.cmd", "c1", &errMsg, nil)
	err := rl.handleMessage(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotErr == nil || gotErr.Error() != "remote error" {
		t.Errorf("expected remote error, got %v", gotErr)
	}
}

func TestReplyListener_HandleMessage_WithError_CallbackFails(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	log := newRecordingLogger()
	rl := NewReplyListener(br, log)
	rl.OnResult("test.cmd", stubResultDeserializer,
		ResultCallbackFunc(func(_ context.Context, _ commands.Result, _ error) error {
			return errors.New("callback failed")
		}),
	)
	errMsg := "remote error"
	data := makeResultEnvelopeBytes("test.cmd", "c1", &errMsg, nil)
	err := rl.handleMessage(context.Background(), data)
	if err != nil {
		t.Fatalf("expected nil (error logged), got %v", err)
	}
	if !log.hasError("reply listener: result handler error") {
		t.Error("expected result handler error log")
	}
}

func TestReplyListener_HandleMessage_SuccessResult(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	rl := NewReplyListener(br, nil)
	var gotResult commands.Result
	rl.OnResult("test.cmd", stubResultDeserializer,
		ResultCallbackFunc(func(_ context.Context, r commands.Result, _ error) error {
			gotResult = r
			return nil
		}),
	)
	result := &stubResult{BaseResult: commands.BaseResult{Name: "stub-result"}, Value: "hello"}
	data := makeResultEnvelopeBytes("test.cmd", "c1", nil, result)
	err := rl.handleMessage(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotResult == nil {
		t.Fatal("expected result")
	}
}

func TestReplyListener_HandleMessage_SuccessResult_WithStruct(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	rl := NewReplyListener(br, nil)
	cb := &structResultCallback{}
	rl.OnResult("test.cmd", stubResultDeserializer, cb)
	result := &stubResult{BaseResult: commands.BaseResult{Name: "stub-result"}, Value: "hello"}
	data := makeResultEnvelopeBytes("test.cmd", "c1", nil, result)
	err := rl.handleMessage(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cb.called {
		t.Error("expected struct callback to be called")
	}
	if cb.gotResult == nil {
		t.Fatal("expected result in struct callback")
	}
}

func TestReplyListener_HandleMessage_ErrorResult_WithStruct(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	rl := NewReplyListener(br, nil)
	cb := &structResultCallback{}
	rl.OnResult("test.cmd", stubResultDeserializer, cb)
	errMsg := "remote failure"
	data := makeResultEnvelopeBytes("test.cmd", "c1", &errMsg, nil)
	err := rl.handleMessage(context.Background(), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cb.called {
		t.Error("expected struct callback to be called")
	}
	if cb.gotErr == nil || cb.gotErr.Error() != "remote failure" {
		t.Errorf("expected remote failure, got %v", cb.gotErr)
	}
}

func TestReplyListener_HandleMessage_StructCallback_ReturnsError(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	rl := NewReplyListener(br, nil)
	cb := &structResultCallback{shouldFail: true}
	rl.OnResult("test.cmd", stubResultDeserializer, cb)
	result := &stubResult{BaseResult: commands.BaseResult{Name: "r"}, Value: "v"}
	data := makeResultEnvelopeBytes("test.cmd", "c1", nil, result)
	err := rl.handleMessage(context.Background(), data)
	if err == nil {
		t.Fatal("expected error from struct callback")
	}
	if !errContains(err, "struct callback error") {
		t.Errorf("expected struct callback error, got %v", err)
	}
}

func TestReplyListener_HandleMessage_DeserializeFail(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	log := newRecordingLogger()
	rl := NewReplyListener(br, log)
	rl.OnResult("test.cmd", failResultDeserializer,
		ResultCallbackFunc(func(_ context.Context, _ commands.Result, _ error) error {
			return nil
		}),
	)
	result := &stubResult{BaseResult: commands.BaseResult{Name: "r"}, Value: "v"}
	data := makeResultEnvelopeBytes("test.cmd", "c1", nil, result)
	err := rl.handleMessage(context.Background(), data)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if !log.hasError("reply listener: deserialize result failed") {
		t.Error("expected deserialize error log")
	}
}

func TestReplyListener_HandleMessage_CallbackError(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	rl := NewReplyListener(br, nil)
	rl.OnResult("test.cmd", stubResultDeserializer,
		ResultCallbackFunc(func(_ context.Context, _ commands.Result, _ error) error {
			return errors.New("callback error")
		}),
	)
	result := &stubResult{BaseResult: commands.BaseResult{Name: "r"}, Value: "v"}
	data := makeResultEnvelopeBytes("test.cmd", "c1", nil, result)
	err := rl.handleMessage(context.Background(), data)
	if err == nil {
		t.Fatal("expected error from callback")
	}
	if !errContains(err, "callback error") {
		t.Errorf("expected callback error, got %v", err)
	}
}

func TestReplyListener_Run_ContextCancel(t *testing.T) {
	t.Parallel()
	br := newStubBroker()
	rl := NewReplyListener(br, nil, WithListenerServiceName("svc"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := rl.Run(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestReplyListener_Run_ConsumesFromCorrectTopic(t *testing.T) {
	t.Parallel()
	cb := newCallbackBroker()
	var consumedTopic string
	var mu sync.Mutex
	cb.consumeFn = func(ctx context.Context, topic string, _ func([]byte) error) error {
		mu.Lock()
		consumedTopic = topic
		mu.Unlock()
		<-ctx.Done()
		return ctx.Err()
	}
	rl := NewReplyListener(cb, nil, WithListenerServiceName("test-svc"))
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_ = rl.Run(ctx)
	mu.Lock()
	defer mu.Unlock()
	expected := "replies.test-svc"
	if consumedTopic != expected {
		t.Errorf("expected topic %q, got %q", expected, consumedTopic)
	}
}
